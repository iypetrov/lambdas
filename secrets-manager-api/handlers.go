package main

import (
	"embed"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/go-chi/chi/v5"

	"github.com/iypetrov/lambdas/secrets-manager-api/clusters"
	"github.com/iypetrov/lambdas/secrets-manager-api/config"
	"github.com/iypetrov/lambdas/secrets-manager-api/logger"
	"github.com/iypetrov/lambdas/secrets-manager-api/secrets"
	"github.com/iypetrov/lambdas/secrets-manager-api/status"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/components"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/views"
	"github.com/iypetrov/lambdas/secrets-manager-api/utils"
)

//go:embed static
var staticFS embed.FS

type RouterHandler struct {
	config         config.Config
	log            logger.Logger
	secretsService *secrets.Service
	clusterService *clusters.Service
}

func (hnd *RouterHandler) StaticFiles() http.Handler {
	return http.StripPrefix("/static", http.FileServer(http.Dir("static")))
}

func (hnd *RouterHandler) HomeView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cluster := r.URL.Query().Get("cluster")
	stats, err := hnd.secretsService.GetStatistics(ctx, cluster)
	if err != nil {
		hnd.log.Error("Failed to get statistics: %v", err)
		stats = &secrets.Statistics{}
	}

	utils.Render(w, r, views.DashboardPage(stats, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) StaticSecretsView(w http.ResponseWriter, r *http.Request) {
	utils.Render(w, r, views.StaticSecretsPage(string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) TLSCertificatesView(w http.ResponseWriter, r *http.Request) {
	utils.Render(w, r, views.TLSCertificatesPage(string(hnd.config.App.Env)))
}

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

	utils.Render(w, r, views.StaticSecretDetailPage(details, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) TLSCertificateDetailView(w http.ResponseWriter, r *http.Request) {
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

	utils.Render(w, r, views.TLSCertificateDetailPage(details, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) GetStatistics(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	cluster := r.URL.Query().Get("cluster")
	stats, err := hnd.secretsService.GetStatistics(ctx, cluster)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyStatistics())
	}

	return utils.Render(w, r, views.StatisticsView(stats))
}

func (hnd *RouterHandler) ListClusters(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	clusters, err := hnd.clusterService.GetAllClusters(ctx)
	if err != nil {
		hnd.log.Error("Failed to list clusters: %v", err)
		clusters = []string{}
	}
	return utils.Render(w, r, components.ClusterOptions(clusters))
}

func (hnd *RouterHandler) ListSecrets(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	secretType := r.URL.Query().Get("type")
	cluster := r.URL.Query().Get("cluster")

	var st secrets.SecretType
	if secretType == "tls" {
		st = secrets.SecretTypeTLSCertificate
	} else {
		st = secrets.SecretTypeStaticSecret
	}

	secretsList, err := hnd.secretsService.ListSecrets(ctx, st, cluster)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptySecretsTable())
	}

	if secretType == "tls" {
		return utils.Render(w, r, components.TLSCertificatesTable(secretsList, string(hnd.config.App.Env)))
	} else {
		return utils.Render(w, r, components.StaticSecretsTable(secretsList, string(hnd.config.App.Env)))
	}
}

func (hnd *RouterHandler) CreateSecret(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var req secrets.CreateSecretRequest

	if err := r.ParseForm(); err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return utils.Render(w, r, components.EmptySecretsTable())
	}

	req.Name = r.FormValue("name")
	req.Value = r.FormValue("value")
	req.Cluster = r.FormValue("cluster")
	typeStr := r.FormValue("type")
	if typeStr == "Static Secret" {
		req.Type = secrets.SecretTypeStaticSecret
	} else {
		req.Type = secrets.SecretTypeTLSCertificate
	}

	// // Parse additional tags
	// req.AdditionalTags = make(map[string]string)
	// for key, values := range r.Form {
	// 	if len(values) > 0 && key != "name" && key != "value" && key != "cluster" && key != "type" {
	// 		// Handle form keys like "additional_tags[key]"
	// 		if len(key) > 15 && key[:15] == "additional_tags[" {
	// 			// Extract tag key from form key like "additional_tags[key]"
	// 			endIdx := len(key) - 1
	// 			if endIdx > 15 {
	// 				tagKey := key[15:endIdx]
	// 				req.AdditionalTags[tagKey] = values[0]
	// 			}
	// 		}
	// 	}
	// }

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
			return utils.Render(w, r, components.EmptySecretsTable())
		}
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Secret '%s' created successfully", resp.Name),
		StatusCode: http.StatusCreated,
	})

	if req.Type == secrets.SecretTypeTLSCertificate {
		return utils.Render(w, r, components.TLSCertificatesTable(secretsList, string(hnd.config.App.Env)))
	} else {
		return utils.Render(w, r, components.StaticSecretsTable(secretsList, string(hnd.config.App.Env)))
	}
}

func (hnd *RouterHandler) GetSecret(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	secretName := r.URL.Query().Get("name")
	if secretName == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("secret name is required")))
		return utils.Render(w, r, components.EmptyModal())
	}

	resp, err := hnd.secretsService.GetSecret(ctx, secretName)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyModal())
	}

	return utils.Render(w, r, components.SecretValueModal(resp.Value, secretName))
}

func (hnd *RouterHandler) UpdateSecret(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return utils.Render(w, r, components.EmptySecretsTable())
	}

	req := secrets.UpdateSecretRequest{
		Name:  r.FormValue("name"),
		Value: r.FormValue("value"),
	}

	resp, err := hnd.secretsService.UpdateSecret(ctx, req)
	if err != nil {
		if contains(err.Error(), "not found") {
			status.AddToast(w, status.ErrorNotFound(err))
			return utils.Render(w, r, components.EmptySecretsTable())
		}
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptySecretsTable())
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Secret '%s' updated successfully", resp.Name),
		StatusCode: http.StatusOK,
	})
	return nil
}

func (hnd *RouterHandler) DeleteSecret(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	secretName := r.URL.Query().Get("name")
	if secretName == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("secret name is required")))
		return utils.Render(w, r, components.EmptySecretsTable())
	}

	cluster, secretType, err := parseSecretName(secretName)
	if err != nil {
		cluster = r.URL.Query().Get("cluster")
		typeStr := r.URL.Query().Get("type")
		if typeStr == "tls" {
			secretType = secrets.SecretTypeTLSCertificate
		} else {
			secretType = secrets.SecretTypeStaticSecret
		}
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
			return utils.Render(w, r, components.EmptySecretsTable())
		}
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Secret '%s' deleted successfully", resp.Name),
		StatusCode: http.StatusOK,
	})

	if secretType == secrets.SecretTypeTLSCertificate {
		return utils.Render(w, r, components.TLSCertificatesTable(secretsList, string(hnd.config.App.Env)))
	} else {
		return utils.Render(w, r, components.StaticSecretsTable(secretsList, string(hnd.config.App.Env)))
	}
}

func (hnd *RouterHandler) AddTag(w http.ResponseWriter, r *http.Request) error {
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

	_, err := hnd.secretsService.AddTag(ctx, req)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return nil
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Tag '%s' added successfully", req.Key),
		StatusCode: http.StatusOK,
	})

	// Determine redirect URL based on secret type
	encodedName := base64.URLEncoding.EncodeToString([]byte(req.Name))
	// Check if it's a TLS certificate by checking the name pattern
	var redirectPath string
	if strings.Contains(req.Name, "TLS_CERTIFICATE") {
		redirectPath = fmt.Sprintf("/%s/p/tls-certificates/%s", hnd.config.App.Env, encodedName)
	} else {
		redirectPath = fmt.Sprintf("/%s/p/static-secrets/%s", hnd.config.App.Env, encodedName)
	}

	utils.HxRedirect(w, redirectPath)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
	return nil
}

func (hnd *RouterHandler) RemoveTag(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	secretName := r.URL.Query().Get("name")
	tagKey := r.URL.Query().Get("key")

	if secretName == "" || tagKey == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("name and key are required")))
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

	// Determine redirect URL based on secret type
	encodedName := base64.URLEncoding.EncodeToString([]byte(secretName))
	// Check if it's a TLS certificate by checking the name pattern
	var redirectPath string
	if strings.Contains(secretName, "TLS_CERTIFICATE") {
		redirectPath = fmt.Sprintf("/%s/p/tls-certificates/%s", hnd.config.App.Env, encodedName)
	} else {
		redirectPath = fmt.Sprintf("/%s/p/static-secrets/%s", hnd.config.App.Env, encodedName)
	}

	utils.HxRedirect(w, redirectPath)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr)))
}

// parseSecretName extracts cluster and type from a secret name
// Format: RESTRICTED.<cluster>.<TYPE>.<name>
// Returns: cluster, secretType, error
func parseSecretName(secretName string) (string, secrets.SecretType, error) {
	parts := strings.Split(secretName, ".")
	if len(parts) < 4 || parts[0] != "RESTRICTED" {
		return "", "", fmt.Errorf("invalid secret name format")
	}

	cluster := parts[1]
	typePart := parts[2]

	var secretType secrets.SecretType
	if typePart == "STATIC_SECRET" {
		secretType = secrets.SecretTypeStaticSecret
	} else if typePart == "TLS_CERTIFICATE" {
		secretType = secrets.SecretTypeTLSCertificate
	} else {
		return "", "", fmt.Errorf("unknown secret type: %s", typePart)
	}

	return cluster, secretType, nil
}
