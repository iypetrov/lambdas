package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/go-chi/chi/v5"

	"github.com/iypetrov/lambdas/secrets-manager-api/secrets"
	"github.com/iypetrov/lambdas/secrets-manager-api/status"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/components"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/views"
	"github.com/iypetrov/lambdas/secrets-manager-api/utils"
)

func (hnd *RouterHandler) TLSCertificateDetailView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	encodedName := chi.URLParam(r, "name")
	if encodedName == "" {
		http.Error(w, "Certificate name is required", http.StatusBadRequest)
		return
	}

	// Base64 decode the secret name
	decoded, err := base64.URLEncoding.DecodeString(encodedName)
	if err != nil {
		http.Error(w, "Invalid certificate name encoding", http.StatusBadRequest)
		return
	}
	secretName := string(decoded)

	details, err := hnd.secretsService.GetSecretDetails(ctx, secretName)
	if err != nil {
		hnd.log.Error("Failed to get certificate details: %v", err)
		http.Error(w, "Failed to retrieve certificate details", http.StatusInternalServerError)
		return
	}

	// Don't fetch certificate value for security - it won't be displayed or pre-filled
	utils.Render(w, r, views.TLSCertificateDetailPage(details, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) GetTLSCertificate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	encodedName := chi.URLParam(r, "name")
	if encodedName == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("certificate name is required")))
		return utils.Render(w, r, components.EmptyModal())
	}

	// Base64 decode the secret name
	decoded, err := base64.URLEncoding.DecodeString(encodedName)
	if err != nil {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("invalid certificate name encoding")))
		return utils.Render(w, r, components.EmptyModal())
	}
	secretName := string(decoded)

	resp, err := hnd.secretsService.GetSecret(ctx, secretName)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyModal())
	}

	return utils.Render(w, r, components.SecretValueModal(resp.Value, secretName))
}

func (hnd *RouterHandler) ListTLSCertificates(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	cluster := r.URL.Query().Get("cluster")

	secretsList, err := hnd.secretsService.ListSecrets(ctx, secrets.SecretTypeTLSCertificate, cluster)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptySecretsTable())
	}

	return utils.Render(w, r, components.TLSCertificatesTable(secretsList, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) CreateTLSCertificate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var req secrets.CreateSecretRequest

	if err := r.ParseForm(); err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return utils.Render(w, r, components.EmptySecretsTable())
	}

	req.Name = r.FormValue("name")
	req.Cluster = r.FormValue("cluster")
	req.Type = secrets.SecretTypeTLSCertificate

	// For TLS certificates, combine crt and key fields
	crt := r.FormValue("crt")
	key := r.FormValue("key")
	if crt == "" || key == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("both certificate (crt) and private key (key) are required for TLS certificates")))
		return utils.Render(w, r, components.EmptySecretsTable())
	}
	// Combine certificate and key with newline separator (standard PEM format)
	req.Value = strings.TrimSpace(crt) + "\n" + strings.TrimSpace(key)

	if req.Cluster == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("cluster is required")))
		return utils.Render(w, r, components.EmptySecretsTable())
	}

	resp, err := hnd.secretsService.CreateSecret(ctx, req)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptySecretsTable())
	}

	// Wait for secret to be available and then for it to appear in the list
	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.InitialInterval = 1 * time.Second
	backoffConfig.MaxInterval = 5 * time.Second
	backoffConfig.Reset()

	// First, wait for GetSecret to succeed (secret is available)
	retryableOperationGetSecret := func() (struct{}, error) {
		_, err := hnd.secretsService.GetSecret(ctx, resp.Name)
		if err == nil {
			return struct{}{}, nil
		}
		// If it's a "not found" error, retry (secret not yet synced)
		if contains(err.Error(), "not found") {
			return struct{}{}, err
		}
		// For other errors, stop retrying
		return struct{}{}, backoff.Permanent(err)
	}

	_, err = backoff.Retry(ctx, retryableOperationGetSecret, backoff.WithBackOff(backoffConfig))
	if err != nil {
		hnd.log.Warn("Certificate created but not yet available after retries: %v", err)
	}

	// Now wait for the secret to appear in the list (tags need to sync)
	backoffConfig.Reset()
	retryableOperationListSecrets := func() ([]secrets.Secret, error) {
		secretsList, err := hnd.secretsService.ListSecrets(ctx, req.Type, req.Cluster)
		if err != nil {
			return nil, backoff.Permanent(err)
		}
		// Check if the newly created secret is in the list
		for _, secret := range secretsList {
			if secret.Name == resp.Name {
				return secretsList, nil
			}
		}
		// Secret not in list yet, retry
		return nil, fmt.Errorf("certificate not yet in list")
	}

	secretsList, err := backoff.Retry(ctx, retryableOperationListSecrets, backoff.WithBackOff(backoffConfig))
	if err != nil {
		hnd.log.Warn("Certificate created but not yet in list after retries: %v", err)
		// Fallback: get the list anyway (might be stale)
		secretsList, err = hnd.secretsService.ListSecrets(ctx, req.Type, req.Cluster)
		if err != nil {
			status.AddToast(w, status.ErrorInternalServerError(err))
			return utils.Render(w, r, components.EmptySecretsTable())
		}
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Certificate '%s' created successfully", resp.Name),
		StatusCode: http.StatusCreated,
	})

	return utils.Render(w, r, components.TLSCertificatesTable(secretsList, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) UpdateTLSCertificate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return nil
	}

	secretName := r.FormValue("name")
	crt := r.FormValue("crt")
	key := r.FormValue("key")

	// For TLS certificates, both crt and key are required
	if crt == "" || key == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("both certificate (crt) and private key (key) are required for TLS certificates")))
		return nil
	}
	value := strings.TrimSpace(crt) + "\n" + strings.TrimSpace(key)

	req := secrets.UpdateSecretRequest{
		Name:  secretName,
		Value: value,
	}

	resp, err := hnd.secretsService.UpdateSecret(ctx, req)
	if err != nil {
		if contains(err.Error(), "not found") {
			status.AddToast(w, status.ErrorNotFound(err))
			return nil
		}
		status.AddToast(w, status.ErrorInternalServerError(err))
		return nil
	}

	// Get updated details to render the page
	details, err := hnd.secretsService.GetSecretDetails(ctx, secretName)
	if err != nil {
		hnd.log.Error("Failed to get certificate details after update: %v", err)
		status.AddToast(w, status.Toast{
			Message:    fmt.Sprintf("Certificate '%s' updated successfully", resp.Name),
			StatusCode: http.StatusOK,
		})
		// Still redirect even if we can't get details
		encodedName := base64.URLEncoding.EncodeToString([]byte(secretName))
		redirectPath := fmt.Sprintf("/%s/p/tls-certificates/%s", hnd.config.App.Env, encodedName)
		utils.HxRedirect(w, redirectPath)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte{})
		return nil
	}

	// Add toast and render the page - the HX-Trigger header should work with HTMX
	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Certificate '%s' updated successfully", resp.Name),
		StatusCode: http.StatusOK,
	})

	return utils.Render(w, r, views.TLSCertificateDetailPage(details, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) DeleteTLSCertificate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	secretName := r.URL.Query().Get("name")
	if secretName == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("certificate name is required")))
		return utils.Render(w, r, components.EmptySecretsTable())
	}

	cluster, secretType, err := parseSecretName(secretName)
	if err != nil {
		cluster = r.URL.Query().Get("cluster")
		secretType = secrets.SecretTypeTLSCertificate
	}

	resp, err := hnd.secretsService.DeleteSecret(ctx, secretName)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptySecretsTable())
	}

	// Wait for secret to be deleted and then for it to disappear from the list
	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.InitialInterval = 500 * time.Millisecond
	backoffConfig.MaxInterval = 5 * time.Second
	backoffConfig.Reset()

	// First, wait for GetSecret to fail (secret is deleted)
	retryableOperationGetSecret := func() (struct{}, error) {
		_, err := hnd.secretsService.GetSecret(ctx, secretName)
		if err != nil {
			// Secret is deleted if it's not found or marked for deletion
			if contains(err.Error(), "not found") || contains(err.Error(), "marked for deletion") {
				return struct{}{}, nil
			}
			return struct{}{}, backoff.Permanent(err)
		}
		// Secret still exists, retry
		return struct{}{}, fmt.Errorf("certificate still exists")
	}

	_, err = backoff.Retry(ctx, retryableOperationGetSecret, backoff.WithBackOff(backoffConfig))
	if err != nil {
		hnd.log.Warn("Certificate deleted but still accessible after retries: %v", err)
	}

	// Now wait for the secret to disappear from the list
	backoffConfig.Reset()
	retryableOperationListSecrets := func() ([]secrets.Secret, error) {
		secretsList, err := hnd.secretsService.ListSecrets(ctx, secretType, cluster)
		if err != nil {
			return nil, backoff.Permanent(err)
		}
		// Check if the deleted secret is still in the list
		for _, secret := range secretsList {
			if secret.Name == secretName {
				// Secret still in list, retry
				return nil, fmt.Errorf("certificate still in list")
			}
		}
		// Secret not in list anymore, success
		return secretsList, nil
	}

	secretsList, err := backoff.Retry(ctx, retryableOperationListSecrets, backoff.WithBackOff(backoffConfig))
	if err != nil {
		hnd.log.Warn("Certificate deleted but still in list after retries: %v", err)
		// Fallback: get the list anyway (might be stale)
		secretsList, err = hnd.secretsService.ListSecrets(ctx, secretType, cluster)
		if err != nil {
			status.AddToast(w, status.ErrorInternalServerError(err))
			return utils.Render(w, r, components.EmptySecretsTable())
		}
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Certificate '%s' deleted successfully", resp.Name),
		StatusCode: http.StatusOK,
	})

	return utils.Render(w, r, components.TLSCertificatesTable(secretsList, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) AddTLSCertificateTag(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return nil
	}

	req := secrets.AddTagRequest{
		Name:  r.FormValue("name"),
		Key:   r.FormValue("key"),
		Value: r.FormValue("value"),
	}

	if req.Name == "" || req.Key == "" || req.Value == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("name, key, and value are required")))
		return nil
	}

	// Prevent adding/overwriting protected tags
	if secrets.IsProtectedTag(req.Key) {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("cannot modify protected tag '%s'", req.Key)))
		return nil
	}

	_, err := hnd.secretsService.AddTag(ctx, req)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return nil
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Tag '%s' added successfully", req.Key),
		StatusCode: http.StatusOK,
	})

	// Determine redirect URL
	encodedName := base64.URLEncoding.EncodeToString([]byte(req.Name))
	redirectPath := fmt.Sprintf("/%s/p/tls-certificates/%s", hnd.config.App.Env, encodedName)

	utils.HxRedirect(w, redirectPath)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
	return nil
}

func (hnd *RouterHandler) RemoveTLSCertificateTag(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	secretName := r.URL.Query().Get("name")
	tagKey := r.URL.Query().Get("key")

	if secretName == "" || tagKey == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("name and key are required")))
		return nil
	}

	// Prevent deletion of protected tags
	if secrets.IsProtectedTag(tagKey) {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("cannot delete protected tag '%s'", tagKey)))
		return nil
	}

	req := secrets.RemoveTagRequest{
		Name: secretName,
		Key:  tagKey,
	}

	_, err := hnd.secretsService.RemoveTag(ctx, req)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return nil
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Tag '%s' removed successfully", tagKey),
		StatusCode: http.StatusOK,
	})

	// Determine redirect URL
	encodedName := base64.URLEncoding.EncodeToString([]byte(secretName))
	redirectPath := fmt.Sprintf("/%s/p/tls-certificates/%s", hnd.config.App.Env, encodedName)

	utils.HxRedirect(w, redirectPath)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
	return nil
}

