package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/iypetrov/lambdas/secrets-manager-api/config"
	"github.com/iypetrov/lambdas/secrets-manager-api/utils"
)

func NewRouter(hnd RouterHandler) *chi.Mux {
    mux := chi.NewRouter()

	if hnd.config.App.Env == config.Local {
		mux.Handle("/static/*", hnd.StaticFiles())
	}

	mux.With().Route(fmt.Sprintf("/%s/p", hnd.config.App.Env), func(mux chi.Router) {
		mux.Get("/home", hnd.HomeView)
	})

	mux.Route(fmt.Sprintf("/%s/api", hnd.config.App.Env), func(mux chi.Router) {
		mux.Route("/v0", func(mux chi.Router) {
			mux.Get("/statistics", utils.MakeTemplHandler(hnd.GetStatistics))
			mux.Get("/clusters", utils.MakeTemplHandler(hnd.ListClusters))
			mux.Route("/secrets", func(mux chi.Router) {
				mux.Get("/", utils.MakeTemplHandler(hnd.ListSecrets))
				mux.Post("/", utils.MakeTemplHandler(hnd.CreateSecret))
				mux.Get("/get", utils.MakeTemplHandler(hnd.GetSecret))
				mux.Put("/", utils.MakeTemplHandler(hnd.UpdateSecret))
				mux.Delete("/", utils.MakeTemplHandler(hnd.DeleteSecret))
			})
		})
	})

	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, fmt.Sprintf("/%s/p/home", hnd.config.App.Env), http.StatusFound)
	})

    return mux
}
