package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/iypetrov/lambdas/secrets-manager-api/utils"
)

func NewRouter(hnd RouterHandler) *chi.Mux {
    mux := chi.NewRouter()
	mux.Handle("/static/*", hnd.StaticFiles())
	mux.With().Route("/p", func(mux chi.Router) {
		mux.Get("/home", hnd.HomeView)
	})

	mux.Route("/api", func(mux chi.Router) {
		mux.Route("/v0", func(mux chi.Router) {
			mux.Route("/secrets", func(mux chi.Router) {
				mux.Get("/", utils.MakeTemplHandler(hnd.GetAllSecrets))
			})
		})
	})

	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/p/home", http.StatusFound)
	})

    return mux
}
