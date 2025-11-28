package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

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

	err = utils.Render(w, r, views.TLSCertificateDetailPage(details, string(hnd.config.App.Env)))
	if err != nil {
		hnd.log.Error("Failed to render TLS certificate detail page: %v", err)
		status.AddToast(w, status.ErrorInternalServerError(err))
		return
	}
}

func (hnd *RouterHandler) ListTLSCertificates(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	cluster := r.URL.Query().Get("cluster")
	searchName := r.URL.Query().Get("search-name")

	secretsList, err := hnd.secretsService.ListSecrets(
		ctx,
		secrets.SecretTypeTLSCertificate,
		cluster,
	)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	// Collect all unique tag key-value pairs from the unfiltered list for the dropdown
	allTagPairs := getAllTagPairs(secretsList)

	// Filter by name if provided
	if searchName != "" {
		filtered := []secrets.Secret{}
		searchNameLower := strings.ToLower(searchName)
		for _, secret := range secretsList {
			if strings.Contains(strings.ToLower(secret.Name), searchNameLower) {
				filtered = append(filtered, secret)
			}
		}
		secretsList = filtered
	}

	// Filter by tag key-value pairs
	// Parse search-tag-key and search-tag-value parameters
	searchTagFilters := make(map[string]string)
	for key, values := range r.URL.Query() {
		if strings.HasPrefix(key, "search-tag-key[") && strings.HasSuffix(key, "]") {
			tagKey := strings.TrimPrefix(key, "search-tag-key[")
			tagKey = strings.TrimSuffix(tagKey, "]")
			if len(values) > 0 && values[0] != "" {
				valueKey := "search-tag-value[" + tagKey + "]"
				if tagValue := r.URL.Query().Get(valueKey); tagValue != "" {
					searchTagFilters[values[0]] = tagValue
				}
			}
		}
	}

	// Apply tag filters
	if len(searchTagFilters) > 0 {
		filtered := []secrets.Secret{}
		for _, secret := range secretsList {
			matched := true
			for filterKey, filterValue := range searchTagFilters {
				if secretValue, exists := secret.Tags[filterKey]; !exists || secretValue != filterValue {
					matched = false
					break
				}
			}
			if matched {
				filtered = append(filtered, secret)
			}
		}
		secretsList = filtered
	}

	return utils.Render(w, r, components.TLSCertificatesTable(secretsList, string(hnd.config.App.Env), searchName, searchTagFilters, allTagPairs))
}

func (hnd *RouterHandler) ImportTLSCertificate(w http.ResponseWriter, r *http.Request) error {
	hnd.importTLSCertificateMu.Lock()
	defer hnd.importTLSCertificateMu.Unlock()

	ctx := r.Context()
	var req secrets.CreateSecretRequest

	err := r.ParseForm()
	if err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	name := r.FormValue("name")
	cluster := r.FormValue("cluster")
	crt := r.FormValue("crt")
	key := r.FormValue("key")

	if cluster == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("cluster is required")))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	if crt == "" || key == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("both certificate (crt) and private key (key) are required for TLS certificates")))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	additionalTags := parseAdditionalTags(r)
	if _, exists := additionalTags["ExpiresAt"]; !exists {
		additionalTags["ExpiresAt"] = time.Now().UTC().Format(time.DateOnly)
	}

	req.Name = name
	req.Cluster = cluster
	req.Type = secrets.SecretTypeTLSCertificate
	req.AdditionalTags = additionalTags

	data := secrets.TLSCertificateData{
		Crt: strings.TrimSpace(crt),
		Key: strings.TrimSpace(key),
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

	_, retryErr := utils.BackoffRetry(ctx, func() (*secrets.GetSecretResponse, error) {
		res, err := hnd.secretsService.GetSecret(ctx, resp.Name)
		if err != nil {
			return res, err
		}
		if res.Value != req.Value {
			return res, fmt.Errorf("certificate value not yet set correctly")
		}
		return res, nil
	})
	if retryErr != nil {
		hnd.log.Warn("Certificate imported but not yet available after retries: %v", retryErr)
	}

	secretsList, err := utils.BackoffRetry(ctx, func() ([]secrets.Secret, error) {
		list, err := hnd.secretsService.ListSecrets(ctx, req.Type, req.Cluster)
		if err != nil {
			return list, err
		}
		for _, secret := range list {
			if secret.Name == resp.Name {
				return list, nil
			}
		}
		return list, fmt.Errorf("certificate not yet visible in list")
	})
	if err != nil {
		secretsList, err = hnd.secretsService.ListSecrets(ctx, req.Type, req.Cluster)
		if err != nil {
			status.AddToast(w, status.ErrorInternalServerError(err))
			return utils.Render(w, r, components.EmptyTLSCertificatesTable())
		}
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Certificate '%s' imported successfully", resp.Name),
		StatusCode: http.StatusCreated,
	})
	allTagPairs := getAllTagPairs(secretsList)
	return utils.Render(w, r, components.TLSCertificatesTable(secretsList, string(hnd.config.App.Env), "", make(map[string]string), allTagPairs))
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
	crt := r.FormValue("crt")
	key := r.FormValue("key")

	if crt == "" || key == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("both certificate (crt) and private key (key) are required for TLS certificates")))
		return nil
	}

	data := secrets.TLSCertificateData{
		Crt: strings.TrimSpace(crt),
		Key: strings.TrimSpace(key),
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

	_, retryErr := utils.BackoffRetry(ctx, func() (*secrets.GetSecretResponse, error) {
		res, err := hnd.secretsService.GetSecret(ctx, resp.Name)
		if err != nil {
			return res, err
		}
		if res.Value != jsonData {
			return res, fmt.Errorf("certificate value not yet updated")
		}
		return res, nil
	})
	if retryErr != nil {
		hnd.log.Warn("Certificate value updated but validation failed after retries: %v", retryErr)
	}

	details, err := utils.BackoffRetry(ctx, func() (*secrets.GetSecretDetailsResponse, error) {
		det, err := hnd.secretsService.GetSecretDetails(ctx, resp.Name)
		if err != nil {
			return det, err
		}
		return det, nil
	})
	if err != nil {
		details, err = hnd.secretsService.GetSecretDetails(ctx, resp.Name)
		if err != nil {
			status.AddToast(w, status.ErrorInternalServerError(err))
			return nil
		}
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
	secretName := r.URL.Query().Get("name")

	if secretName == "" {
		status.AddToast(w, status.ErrorBadRequest(fmt.Errorf("certificate name is required")))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	resp, err := hnd.secretsService.DeleteSecret(ctx, secretName)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	_, retryErr := utils.BackoffRetry(ctx, func() (*secrets.GetSecretResponse, error) {
		res, err := hnd.secretsService.GetSecret(ctx, resp.Name)
		if err != nil {
			if strings.Contains(err.Error(), "ResourceNotFoundException") {
				return nil, nil
			}
			return res, err
		}
		return res, fmt.Errorf("certificate still exists")
	})
	if retryErr != nil {
		hnd.log.Warn("Certificate deletion validated but GetSecret check failed: %v", retryErr)
	}

	cluster, secretType, err := secrets.ParseSecretName(secretName)
	if err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return utils.Render(w, r, components.EmptyTLSCertificatesTable())
	}

	secretsList, err := utils.BackoffRetry(ctx, func() ([]secrets.Secret, error) {
		list, err := hnd.secretsService.ListSecrets(ctx, secretType, cluster)
		if err != nil {
			return list, err
		}
		for _, secret := range list {
			if secret.Name == resp.Name {
				return list, fmt.Errorf("certificate still visible in list")
			}
		}
		return list, nil
	})
	if err != nil {
		secretsList, err = hnd.secretsService.ListSecrets(ctx, secretType, cluster)
		if err != nil {
			status.AddToast(w, status.ErrorInternalServerError(err))
			return utils.Render(w, r, components.EmptyTLSCertificatesTable())
		}
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Certificate '%s' deleted successfully", resp.Name),
		StatusCode: http.StatusOK,
	})
	allTagPairs := getAllTagPairs(secretsList)
	return utils.Render(w, r, components.TLSCertificatesTable(secretsList, string(hnd.config.App.Env), "", make(map[string]string), allTagPairs))
}

func (hnd *RouterHandler) AddTLSCertificateTag(w http.ResponseWriter, r *http.Request) error {
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

	details, err := hnd.secretsService.GetSecretDetails(ctx, req.Name)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return nil
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Tag '%s' added successfully", key),
		StatusCode: http.StatusOK,
	})
	return utils.Render(w, r, views.TLSCertificateDetailPage(details, string(hnd.config.App.Env)))
}

func (hnd *RouterHandler) RemoveTLSCertificateTag(w http.ResponseWriter, r *http.Request) error {
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

	details, err := hnd.secretsService.GetSecretDetails(ctx, req.Name)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return nil
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Tag '%s' removed successfully", key),
		StatusCode: http.StatusOK,
	})
	return utils.Render(w, r, views.TLSCertificateDetailPage(details, string(hnd.config.App.Env)))
}
