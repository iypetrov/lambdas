package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/iypetrov/lambdas/secrets-manager-api/clusters"
	"github.com/iypetrov/lambdas/secrets-manager-api/config"
	"github.com/iypetrov/lambdas/secrets-manager-api/images"
	"github.com/iypetrov/lambdas/secrets-manager-api/logger"
	"github.com/iypetrov/lambdas/secrets-manager-api/metadata"
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
	imagesService      *images.Service
	metadataService   *metadata.Service

	createStaticSecretMu sync.Mutex
	updateStaticSecretMu sync.Mutex
	deleteStaticSecretMu sync.Mutex

	importTLSCertificateMu sync.Mutex
	updateTLSCertificateMu sync.Mutex
	deleteTLSCertificateMu sync.Mutex
}

func (hnd *RouterHandler) StaticFiles() http.Handler {
	if hnd.config.App.Env == config.Local {
		return http.StripPrefix(fmt.Sprintf("/%s/static", hnd.config.App.Env), http.FileServer(http.Dir("static")))
	}
	return http.StripPrefix(fmt.Sprintf("/%s/static", hnd.config.App.Env), http.HandlerFunc(hnd.serveStaticFromS3))
}

func (hnd *RouterHandler) serveStaticFromS3(w http.ResponseWriter, r *http.Request) {
	prefix := fmt.Sprintf("/%s/static", hnd.config.App.Env)
    filePath := strings.TrimPrefix(r.URL.Path, prefix)

    if filePath == "" || filePath == "/" {
        http.NotFound(w, r)
        return
    }

    if err := hnd.imagesService.ServeStaticFile(w, r, filePath); err != nil {
        http.NotFound(w, r)
        return
    }
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

func (hnd *RouterHandler) ListAuditEvents(w http.ResponseWriter, r *http.Request) error {
	hnd.log.Info("Metadata service is available, fetching audit events")
	ctx := r.Context()
	events, err := hnd.metadataService.GetAuditTLSEvent(ctx)
	if err != nil {
		hnd.log.Error("Failed to get audit events: %v", err)
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyAuditEventsTable())
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].ExpireAt > events[j].ExpireAt
	})

	return utils.Render(w, r, components.AuditEventsTable(events, string(hnd.config.App.Env)))
}
