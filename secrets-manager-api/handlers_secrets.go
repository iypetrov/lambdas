package main

import (
	"fmt"
	"net/http"

	"github.com/cenkalti/backoff/v5"
	"github.com/go-chi/chi/v5"

	"github.com/iypetrov/lambdas/secrets-manager-api/secrets"
	"github.com/iypetrov/lambdas/secrets-manager-api/status"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/components"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/views"
	"github.com/iypetrov/lambdas/secrets-manager-api/utils"
)

func (hnd *RouterHandler) StaticSecretDetailView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	secretName, err := utils.Base64Decode(chi.URLParam(r, "name"))
	if err != nil {
		http.Error(w, "Invalid secret name encoding", http.StatusBadRequest)
		return
	}

	details, err := hnd.secretsService.GetSecretDetails(ctx, secretName)
	if err != nil {
		hnd.log.Error("Failed to get secret details: %v", err)
		status.AddToast(w, status.ErrorInternalServerError(err))
		return
	}

	utils.Render(w, r, views.StaticSecretDetailPage(details, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) ListStaticSecrets(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	cluster := r.URL.Query().Get("cluster")

	secretsList, err := hnd.secretsService.ListSecrets(
		ctx,
		secrets.SecretTypeStaticSecret,
		cluster,
	)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	return utils.Render(w, r, components.StaticSecretsTable(secretsList, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) CreateStaticSecret(w http.ResponseWriter, r *http.Request) error {
	hnd.createStaticSecretMu.Lock()
	defer hnd.createStaticSecretMu.Unlock()

	ctx := r.Context()
	var req secrets.CreateSecretRequest

	err := r.ParseForm()
	if err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	name := r.FormValue("name")
	cluster := r.FormValue("cluster")
	value := r.FormValue("value")

	if cluster == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("cluster is required")))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	if value == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("value is required")))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	req.Name = name
	req.Cluster = cluster
	req.Type = secrets.SecretTypeStaticSecret
	req.Value = value

	resp, err := hnd.secretsService.CreateSecret(ctx, req)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	_, err = utils.BackoffRetry(ctx, func() (*secrets.GetSecretResponse, error) {
		res, err := hnd.secretsService.GetSecret(ctx, resp.Name)
		if err == nil {
			return res, nil
		}
		return res, backoff.Permanent(err)
	})
	if err != nil {
		hnd.log.Warn("Secret created but not yet available after retries: %v", err)
	}

	secretsList, err := hnd.secretsService.ListSecrets(ctx, req.Type, req.Cluster)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Secret '%s' created successfully", resp.Name),
		StatusCode: http.StatusCreated,
	})
	return utils.Render(w, r, components.StaticSecretsTable(secretsList, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) UpdateStaticSecret(w http.ResponseWriter, r *http.Request) error {
	hnd.updateStaticSecretMu.Lock()
	defer hnd.updateStaticSecretMu.Unlock()

	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return nil
	}

	secretName := r.FormValue("name")
	value := r.FormValue("value")

	if value == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("value is required")))
		return nil
	}

	req := secrets.UpdateSecretRequest{
		Name:  secretName,
		Value: value,
	}

	resp, err := hnd.secretsService.UpdateSecret(ctx, req)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return nil
	}

	_, err = utils.BackoffRetry(ctx, func() (*secrets.GetSecretResponse, error) {
		res, err := hnd.secretsService.GetSecret(ctx, resp.Name)
		if err == nil {
			if res.Value == value {
				return res, nil
			}
			return res, fmt.Errorf("secret value not yet updated")
		}
		return res, backoff.Permanent(err)
	})

	details, err := hnd.secretsService.GetSecretDetails(ctx, resp.Name)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return nil
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Secret '%s' updated successfully", resp.Name),
		StatusCode: http.StatusOK,
	})
	return utils.Render(w, r, views.StaticSecretDetailPage(details, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) DeleteStaticSecret(w http.ResponseWriter, r *http.Request) error {
	hnd.deleteStaticSecretMu.Lock()
	defer hnd.deleteStaticSecretMu.Unlock()

	ctx := r.Context()
	secretName := r.URL.Query().Get("name")

	if secretName == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("secret name is required")))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	resp, err := hnd.secretsService.DeleteSecret(ctx, secretName)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	_, err = utils.BackoffRetry(ctx, func() (*secrets.GetSecretResponse, error) {
		res, err := hnd.secretsService.GetSecret(ctx, resp.Name)
		if err != nil {
			return res, backoff.Permanent(err)
		}
		return res, fmt.Errorf("secret still exists")
	})

	cluster, secretType, err := parseSecretName(secretName)
	if err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	secretsList, err := hnd.secretsService.ListSecrets(ctx, secretType, cluster)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Secret '%s' deleted successfully", resp.Name),
		StatusCode: http.StatusOK,
	})
	return utils.Render(w, r, components.StaticSecretsTable(secretsList, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) AddStaticSecretTag(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return nil
	}

	name := r.FormValue("name")
	key := r.FormValue("key")
	value := r.FormValue("value")

	req := secrets.AddTagRequest{
		Name:  name,
		Key:   key,
		Value: value,
	}

	if req.Name == "" || req.Key == "" || req.Value == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("name, key, and value are required")))
		return nil
	}

	if secrets.IsProtectedTag(req.Key) {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("cannot modify protected tag '%s'", req.Key)))
		return nil
	}

	_, err := hnd.secretsService.AddTag(ctx, req)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return nil
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Tag '%s' added successfully", key),
		StatusCode: http.StatusOK,
	})
	return nil
}

func (hnd *RouterHandler) RemoveStaticSecretTag(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	name := r.URL.Query().Get("name")
	key := r.URL.Query().Get("key")

	if name == "" || key == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("name and key are required")))
		return nil
	}

	if secrets.IsProtectedTag(key) {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("cannot delete protected tag '%s'", key)))
		return nil
	}

	req := secrets.RemoveTagRequest{
		Name: name,
		Key:  key,
	}

	_, err := hnd.secretsService.RemoveTag(ctx, req)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return nil
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Tag '%s' removed successfully", key),
		StatusCode: http.StatusOK,
	})
	return nil
}


