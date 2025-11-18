package main

import (
	"embed"
	"net/http"

	"github.com/iypetrov/lambdas/secrets-manager-api/config"
	"github.com/iypetrov/lambdas/secrets-manager-api/logger"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/views"
	"github.com/iypetrov/lambdas/secrets-manager-api/utils"
)

//go:embed static
var staticFS embed.FS

type RouterHandler struct {
	config        config.Config
	log           logger.Logger
}

func (hnd *RouterHandler) StaticFiles() http.Handler {
	return http.StripPrefix("/static", http.FileServer(http.Dir("static")))
}

func (hnd *RouterHandler) HomeView(w http.ResponseWriter, r *http.Request) {
	utils.Render(w, r, views.HomePage())
}

func (hnd *RouterHandler) GetAllSecrets(w http.ResponseWriter, r *http.Request) error {
	hnd.log.Info("GetAllSecrets called")
	return nil
}
