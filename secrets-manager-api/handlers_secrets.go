package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/go-chi/chi/v5"

	"github.com/iypetrov/lambdas/secrets-manager-api/secrets"
	"github.com/iypetrov/lambdas/secrets-manager-api/status"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/components"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/views"
	"github.com/iypetrov/lambdas/secrets-manager-api/utils"
)

func (hnd *RouterHandler) StaticSecretDetailView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	encodedName := chi.URLParam(r, "name")
	if encodedName == "" {
		http.Error(w, "Secret name is required", http.StatusBadRequest)
		return
	}

	// Base64 decode the secret name
	decoded, err := base64.URLEncoding.DecodeString(encodedName)
	if err != nil {
		http.Error(w, "Invalid secret name encoding", http.StatusBadRequest)
		return
	}
	secretName := string(decoded)

	details, err := hnd.secretsService.GetSecretDetails(ctx, secretName)
	if err != nil {
		hnd.log.Error("Failed to get secret details: %v", err)
		http.Error(w, "Failed to retrieve secret details", http.StatusInternalServerError)
		return
	}

	// Don't fetch secret value for security - it won't be displayed or pre-filled
	utils.Render(w, r, views.StaticSecretDetailPage(details, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) ListStaticSecrets(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	cluster := r.URL.Query().Get("cluster")

	secretsList, err := hnd.secretsService.ListSecrets(ctx, secrets.SecretTypeStaticSecret, cluster)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	return utils.Render(w, r, components.StaticSecretsTable(secretsList, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) CreateStaticSecret(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var req secrets.CreateSecretRequest

	if err := r.ParseForm(); err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	req.Name = r.FormValue("name")
	req.Cluster = r.FormValue("cluster")
	req.Type = secrets.SecretTypeStaticSecret
	req.Value = r.FormValue("value")

	if req.Cluster == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("cluster is required")))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	if req.Value == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("value is required")))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	resp, err := hnd.secretsService.CreateSecret(ctx, req)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
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
		hnd.log.Warn("Secret created but not yet available after retries: %v", err)
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
		return nil, fmt.Errorf("secret not yet in list")
	}

	secretsList, err := backoff.Retry(ctx, retryableOperationListSecrets, backoff.WithBackOff(backoffConfig))
	if err != nil {
		hnd.log.Warn("Secret created but not yet in list after retries: %v", err)
		// Fallback: get the list anyway (might be stale)
		secretsList, err = hnd.secretsService.ListSecrets(ctx, req.Type, req.Cluster)
		if err != nil {
			status.AddToast(w, status.ErrorInternalServerError(err))
			return utils.Render(w, r, components.EmptyStaticSecretsTable())
		}
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Secret '%s' created successfully", resp.Name),
		StatusCode: http.StatusCreated,
	})

	return utils.Render(w, r, components.StaticSecretsTable(secretsList, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) UpdateStaticSecret(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return nil
	}

	secretName := r.FormValue("name")
	value := r.FormValue("value")

	if value == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("value is required")))
		return nil
	}

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
		hnd.log.Error("Failed to get secret details after update: %v", err)
		status.AddToast(w, status.Toast{
			Message:    fmt.Sprintf("Secret '%s' updated successfully", resp.Name),
			StatusCode: http.StatusOK,
		})
		// Still redirect even if we can't get details
		encodedName := base64.URLEncoding.EncodeToString([]byte(secretName))
		redirectPath := fmt.Sprintf("/%s/p/static-secrets/%s", hnd.config.App.Env, encodedName)
		utils.HxRedirect(w, redirectPath)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte{})
		return nil
	}

	// Add toast and render the page - the HX-Trigger header should work with HTMX
	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Secret '%s' updated successfully", resp.Name),
		StatusCode: http.StatusOK,
	})

	return utils.Render(w, r, views.StaticSecretDetailPage(details, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) DeleteStaticSecret(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	secretName := r.URL.Query().Get("name")
	if secretName == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("secret name is required")))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	cluster, secretType, err := parseSecretName(secretName)
	if err != nil {
		cluster = r.URL.Query().Get("cluster")
		secretType = secrets.SecretTypeStaticSecret
	}

	resp, err := hnd.secretsService.DeleteSecret(ctx, secretName)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
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
		return struct{}{}, fmt.Errorf("secret still exists")
	}

	_, err = backoff.Retry(ctx, retryableOperationGetSecret, backoff.WithBackOff(backoffConfig))
	if err != nil {
		hnd.log.Warn("Secret deleted but still accessible after retries: %v", err)
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
				return nil, fmt.Errorf("secret still in list")
			}
		}
		// Secret not in list anymore, success
		return secretsList, nil
	}

	secretsList, err := backoff.Retry(ctx, retryableOperationListSecrets, backoff.WithBackOff(backoffConfig))
	if err != nil {
		hnd.log.Warn("Secret deleted but still in list after retries: %v", err)
		// Fallback: get the list anyway (might be stale)
		secretsList, err = hnd.secretsService.ListSecrets(ctx, secretType, cluster)
		if err != nil {
			status.AddToast(w, status.ErrorInternalServerError(err))
			return utils.Render(w, r, components.EmptyStaticSecretsTable())
		}
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Secret '%s' deleted successfully", resp.Name),
		StatusCode: http.StatusOK,
	})

	return utils.Render(w, r, components.StaticSecretsTable(secretsList, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) AddStaticSecretTag(w http.ResponseWriter, r *http.Request) error {
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
	redirectPath := fmt.Sprintf("/%s/p/static-secrets/%s", hnd.config.App.Env, encodedName)

	utils.HxRedirect(w, redirectPath)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
	return nil
}

func (hnd *RouterHandler) RemoveStaticSecretTag(w http.ResponseWriter, r *http.Request) error {
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
	redirectPath := fmt.Sprintf("/%s/p/static-secrets/%s", hnd.config.App.Env, encodedName)

	utils.HxRedirect(w, redirectPath)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
	return nil
}

