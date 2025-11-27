package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
	"strings"

	"github.com/cenkalti/backoff/v5"
	"github.com/go-chi/chi/v5"

	"github.com/iypetrov/lambdas/secrets-manager-api/secrets"
	"github.com/iypetrov/lambdas/secrets-manager-api/status"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/components"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/views"
	"github.com/iypetrov/lambdas/secrets-manager-api/utils"
)

func (hnd *RouterHandler) TLSCertificateDetailView(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	secretName, err := utils.Base64Decode(chi.URLParam(r, "name"))
	if err != nil {
		http.Error(w, "Invalid certificate name encoding", http.StatusBadRequest)
		return
	}

	details, err := hnd.secretsService.GetSecretDetails(ctx, secretName)
	if err != nil {
		hnd.log.Error("Failed to get certificate details: %v", err)
		status.AddToast(w, status.ErrorInternalServerError(err))
		return
	}

	utils.Render(w, r, views.TLSCertificateDetailPage(details, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) ListTLSCertificates(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	secretsList, err := hnd.secretsService.ListSecrets(
		ctx, 
		secrets.SecretTypeTLSCertificate, 
		r.URL.Query().Get("cluster"),
	)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	return utils.Render(w, r, components.TLSCertificatesTable(secretsList, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) CreateTLSCertificate(w http.ResponseWriter, r *http.Request) error {
	hnd.createTLSCertificateMu.Lock()
	defer hnd.createTLSCertificateMu.Unlock()

	ctx := r.Context()
	var req secrets.CreateSecretRequest

	err := r.ParseForm()
	if err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	if r.FormValue("cluster") == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("cluster is required")))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	if r.FormValue("crt") == "" || r.FormValue("key") == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("both certificate (crt) and private key (key) are required for TLS certificates")))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	req.Name = r.FormValue("name")
	req.Cluster = r.FormValue("cluster")
	req.Type = secrets.SecretTypeTLSCertificate
	req.AdditionalTags = map[string]string{
		"ExpiresAt": time.Now().UTC().Format(time.DateOnly),
	}

	data := secrets.TLSCertificateData{
		Crt: strings.TrimSpace(r.FormValue("crt")),
		Key: strings.TrimSpace(r.FormValue("key")),
	}
	jsonData, err := utils.JsonMarshal(data)
	if err != nil {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("failed to encode TLS certificate data: %v", err)))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}
	req.Value = jsonData 

	resp, err := hnd.secretsService.CreateSecret(ctx, req)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	_, err = utils.BackoffRetry(ctx, func() (*secrets.GetSecretResponse, error) {
		res, err := hnd.secretsService.GetSecret(ctx, resp.Name)
		if err == nil {
			return res, nil
		}
		return res, backoff.Permanent(err)
	})

	secretsList, err := hnd.secretsService.ListSecrets(ctx, req.Type, req.Cluster)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Certificate '%s' created successfully", resp.Name),
		StatusCode: http.StatusCreated,
	})
	return utils.Render(w, r, components.TLSCertificatesTable(secretsList, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) UpdateTLSCertificate(w http.ResponseWriter, r *http.Request) error {
	hnd.updateTLSCertificateMu.Lock()
	defer hnd.updateTLSCertificateMu.Unlock()

	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return nil
	}

	secretName := r.FormValue("name")

	if r.FormValue("crt") == "" || r.FormValue("key") == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("both certificate (crt) and private key (key) are required for TLS certificates")))
		return nil
	}

	data := secrets.TLSCertificateData{
		Crt: strings.TrimSpace(r.FormValue("crt")),
		Key: strings.TrimSpace(r.FormValue("key")),
	}
	jsonData, err := utils.JsonMarshal(data)
	if err != nil {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("failed to encode TLS certificate data: %v", err)))
		return nil
	}

	req := secrets.UpdateSecretRequest{
		Name:  secretName,
		Value: jsonData,
	}
	resp, err := hnd.secretsService.UpdateSecret(ctx, req)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return nil
	}

	_, err = utils.BackoffRetry(ctx, func() (*secrets.GetSecretResponse, error) {
		res, err := hnd.secretsService.GetSecret(ctx, resp.Name)
		if err == nil {
			return res, nil
		}
		if res.Value == jsonData {
			return res, nil
		}
		return res, backoff.Permanent(err)
	})

	details, err := hnd.secretsService.GetSecretDetails(ctx, resp.Name)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return nil
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Certificate '%s' updated successfully", resp.Name),
		StatusCode: http.StatusOK,
	})
	return utils.Render(w, r, views.TLSCertificateDetailPage(details, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) DeleteTLSCertificate(w http.ResponseWriter, r *http.Request) error {
	hnd.deleteTLSCertificateMu.Lock()
	defer hnd.deleteTLSCertificateMu.Unlock()

	ctx := r.Context()
	if r.URL.Query().Get("name") == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("certificate name is required")))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	resp, err := hnd.secretsService.DeleteSecret(ctx, r.URL.Query().Get("name"))
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	_, err = utils.BackoffRetry(ctx, func() (*secrets.GetSecretResponse, error) {
		res, err := hnd.secretsService.GetSecret(ctx, resp.Name)
		if err == nil {
			return res, nil
		}
		return res, backoff.Permanent(err)
	})

	cluster, secretType, err := parseSecretName(r.URL.Query().Get("name"))
	if err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	secretsList, err := hnd.secretsService.ListSecrets(ctx, secretType, cluster)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Certificate '%s' deleted successfully", resp.Name),
		StatusCode: http.StatusOK,
	})
	return utils.Render(w, r, components.TLSCertificatesTable(secretsList, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) AddTLSCertificateTag(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return nil
	}

	req := secrets.AddTagRequest{
		Name:  r.FormValue("name"),
		Key:   r.FormValue("key"),
		Value: r.FormValue("value"),
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
		Message:    fmt.Sprintf("Tag '%s' added successfully", req.Key),
		StatusCode: http.StatusOK,
	})
	return nil
}

func (hnd *RouterHandler) RemoveTLSCertificateTag(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	secretName := r.URL.Query().Get("name")
	tagKey := r.URL.Query().Get("key")

	if secretName == "" || tagKey == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("name and key are required")))
		return nil
	}

	if secrets.IsProtectedTag(tagKey) {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("cannot delete protected tag '%s'", tagKey)))
		return nil
	}

	req := secrets.RemoveTagRequest{
		Name: secretName,
		Key:  tagKey,
	}

	_, err := hnd.secretsService.RemoveTag(ctx, req)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return nil
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Tag '%s' removed successfully", tagKey),
		StatusCode: http.StatusOK,
	})
	return nil
}
