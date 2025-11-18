package utils

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/iypetrov/lambdas/secrets-manager-api/status"
)

func Render(w http.ResponseWriter, r *http.Request, c templ.Component) error {
	w.Header().Set("Content-Type", "text/html")

	err := c.Render(r.Context(), w)
	if err != nil {
		return status.ErrorInternalServerError(fmt.Errorf("server failed to render this component"))
	}

	return nil
}

func MakeTemplHandler(f func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			var t status.Toast
			if errors.As(err, &t) {
				status.AddToast(w, t)
			}
		}
	}
}

func HxRedirect(w http.ResponseWriter, path string) {
	w.Header().Set("HX-Redirect", path)
}
