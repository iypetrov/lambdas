package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/iypetrov/lambdas/secrets-manager-api/utils"
)

func NewRouter(hnd RouterHandler) *chi.Mux {
    mux := chi.NewRouter()
	mux.Handle(fmt.Sprintf("/%s/static/*", hnd.config.App.Env), hnd.StaticFiles())
	mux.With().Route(fmt.Sprintf("/%s/p", hnd.config.App.Env), func(mux chi.Router) {
		mux.Get("/home", hnd.HomeView)
	})

	mux.Route(fmt.Sprintf("/%s/api", hnd.config.App.Env), func(mux chi.Router) {
		mux.Route("/v0", func(mux chi.Router) {
			mux.Route("/secrets", func(mux chi.Router) {
				mux.Get("/", utils.MakeTemplHandler(hnd.GetAllSecrets))
			})
		})
	})

	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, fmt.Sprintf("/%s/p/home", hnd.config.App.Env), http.StatusFound)
	})

    return mux
}
