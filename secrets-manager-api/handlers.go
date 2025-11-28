package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/iypetrov/lambdas/secrets-manager-api/clusters"
	"github.com/iypetrov/lambdas/secrets-manager-api/config"
	"github.com/iypetrov/lambdas/secrets-manager-api/logger"
	"github.com/iypetrov/lambdas/secrets-manager-api/secrets"
	"github.com/iypetrov/lambdas/secrets-manager-api/status"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/components"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/views"
	"github.com/iypetrov/lambdas/secrets-manager-api/utils"
)


type RouterHandler struct {
	config         config.Config
	log            logger.Logger
	secretsService *secrets.Service
	clusterService *clusters.Service

	createStaticSecretMu sync.Mutex
	updateStaticSecretMu sync.Mutex
	deleteStaticSecretMu sync.Mutex

	importTLSCertificateMu sync.Mutex
	updateTLSCertificateMu sync.Mutex
	deleteTLSCertificateMu sync.Mutex
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

	if err := utils.Render(w, r, views.DashboardPage(stats, string(hnd.config.App.Env))); err != nil {
		hnd.log.Error("Failed to render dashboard page: %v", err)
	}
}

func (hnd *RouterHandler) StaticSecretsView(w http.ResponseWriter, r *http.Request) {
	if err := utils.Render(w, r, views.StaticSecretsPage(string(hnd.config.App.Env))); err != nil {
		hnd.log.Error("Failed to render static secrets page: %v", err)
	}
}

func (hnd *RouterHandler) TLSCertificatesView(w http.ResponseWriter, r *http.Request) {
	if err := utils.Render(w, r, views.TLSCertificatesPage(string(hnd.config.App.Env))); err != nil {
		hnd.log.Error("Failed to render TLS certificates page: %v", err)
	}
}

func (hnd *RouterHandler) GetStatistics(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	cluster := r.URL.Query().Get("cluster")
	stats, err := hnd.secretsService.GetStatistics(ctx, cluster)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyStatistics())
	}

	return utils.Render(w, r, views.StatisticsView(stats, string(hnd.config.App.Env)))
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
	switch typePart {
	case "STATIC_SECRET":
		secretType = secrets.SecretTypeStaticSecret
	case "TLS_CERTIFICATE":
		secretType = secrets.SecretTypeTLSCertificate
	default:
		return "", "", fmt.Errorf("unknown secret type: %s", typePart)
	}

	return cluster, secretType, nil
}
