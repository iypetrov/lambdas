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
		mux.Get("/static-secrets", hnd.StaticSecretsView)
		mux.Get("/static-secrets/{name}", hnd.StaticSecretDetailView)
		mux.Get("/tls-certificates", hnd.TLSCertificatesView)
		mux.Get("/tls-certificates/{name}", hnd.TLSCertificateDetailView)
	})

	mux.Route(fmt.Sprintf("/%s/api", hnd.config.App.Env), func(mux chi.Router) {
		mux.Route("/v0", func(mux chi.Router) {
			mux.Get("/statistics", utils.MakeTemplHandler(hnd.GetStatistics))
			mux.Get("/clusters", utils.MakeTemplHandler(hnd.ListClusters))

			mux.Route("/static-secrets", func(mux chi.Router) {
				mux.Get("/", utils.MakeTemplHandler(hnd.ListStaticSecrets))
				mux.Post("/tags", utils.MakeTemplHandler(hnd.AddStaticSecretTag))
				mux.Post("/", utils.MakeTemplHandler(hnd.CreateStaticSecret))
				mux.Put("/", utils.MakeTemplHandler(hnd.UpdateStaticSecret))
				mux.Delete("/", utils.MakeTemplHandler(hnd.DeleteStaticSecret))
				mux.Delete("/tags", utils.MakeTemplHandler(hnd.RemoveStaticSecretTag))
			})

			mux.Route("/tls-certificates", func(mux chi.Router) {
				mux.Get("/", utils.MakeTemplHandler(hnd.ListTLSCertificates))
				mux.Post("/tags", utils.MakeTemplHandler(hnd.AddTLSCertificateTag))
				mux.Post("/", utils.MakeTemplHandler(hnd.ImportTLSCertificate))
				mux.Put("/", utils.MakeTemplHandler(hnd.UpdateTLSCertificate))
				mux.Delete("/", utils.MakeTemplHandler(hnd.DeleteTLSCertificate))
				mux.Delete("/tags", utils.MakeTemplHandler(hnd.RemoveTLSCertificateTag))
			})
		})
	})

	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, fmt.Sprintf("/%s/p/home", hnd.config.App.Env), http.StatusFound)
	})

	return mux
}
