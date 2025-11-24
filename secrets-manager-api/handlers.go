package main

import (
	"embed"
	"fmt"
	"net/http"

	"github.com/iypetrov/lambdas/secrets-manager-api/config"
	"github.com/iypetrov/lambdas/secrets-manager-api/logger"
	"github.com/iypetrov/lambdas/secrets-manager-api/secrets"
	"github.com/iypetrov/lambdas/secrets-manager-api/clusters"
	"github.com/iypetrov/lambdas/secrets-manager-api/status"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/components"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/views"
	"github.com/iypetrov/lambdas/secrets-manager-api/utils"
)

//go:embed static
var staticFS embed.FS

type RouterHandler struct {
	config        config.Config
	log           logger.Logger
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
	
	// Parse additional tags
	req.AdditionalTags = make(map[string]string)
	for key, values := range r.Form {
		if len(values) > 0 && key != "name" && key != "value" && key != "cluster" && key != "type" {
			// Handle form keys like "additional_tags[key]"
			if len(key) > 15 && key[:15] == "additional_tags[" {
				// Extract tag key from form key like "additional_tags[key]"
				endIdx := len(key) - 1
				if endIdx > 15 {
					tagKey := key[15:endIdx]
					req.AdditionalTags[tagKey] = values[0]
				}
			}
		}
	}

	if req.Cluster == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("cluster is required")))
		return utils.Render(w, r, components.EmptySecretsTable())
	}

	resp, err := hnd.secretsService.CreateSecret(ctx, req)
	if err != nil {
		if contains(err.Error(), "already exists") {
			status.AddToast(w, status.ErrorConflict(err))
			return utils.Render(w, r, components.EmptySecretsTable())
		}
		if contains(err.Error(), "invalid request") {
			status.AddToast(w, status.ErrorBadRequest(err))
			return utils.Render(w, r, components.EmptySecretsTable())
		}
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptySecretsTable())
	}
	
	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Secret '%s' created successfully", resp.Name),
		StatusCode: http.StatusCreated,
	})
	return nil
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
		if contains(err.Error(), "not found") {
			status.AddToast(w, status.ErrorNotFound(err))
			return utils.Render(w, r, components.EmptyModal())
		}
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

	resp, err := hnd.secretsService.DeleteSecret(ctx, secretName)
	if err != nil {
		if contains(err.Error(), "not found") {
			status.AddToast(w, status.ErrorNotFound(err))
			return utils.Render(w, r, components.EmptySecretsTable())
		}
		if contains(err.Error(), "already scheduled") {
			status.AddToast(w, status.ErrorBadRequest(err))
			return utils.Render(w, r, components.EmptySecretsTable())
		}
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptySecretsTable())
	}
	
	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Secret '%s' deleted successfully", resp.Name),
		StatusCode: http.StatusOK,
	})
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr)))
}
