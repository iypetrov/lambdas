package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/iypetrov/lambdas/secrets-manager-api/secrets"
	"github.com/iypetrov/lambdas/secrets-manager-api/status"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/components"
	"github.com/iypetrov/lambdas/secrets-manager-api/templates/views"
	"github.com/iypetrov/lambdas/secrets-manager-api/utils"
)

func parseAdditionalTags(r *http.Request) map[string]string {
	tags := make(map[string]string)
	prefix := "additional_tags["
	
	for key, values := range r.Form {
		if strings.HasPrefix(key, prefix) && strings.HasSuffix(key, "]") {
			tagKey := strings.TrimPrefix(key, prefix)
			tagKey = strings.TrimSuffix(tagKey, "]")
			
			if len(values) > 0 && values[0] != "" {
				tags[tagKey] = values[0]
			}
		}
	}
	
	return tags
}

func getAllTagPairs(secretsList []secrets.Secret) []secrets.TagPair {
	tagPairsMap := make(map[string]bool) // Use "key:value" as key
	var tagPairs []secrets.TagPair
	
	for _, secret := range secretsList {
		for key, value := range secret.Tags {
			pairKey := key + ":" + value
			if !tagPairsMap[pairKey] {
				tagPairsMap[pairKey] = true
				tagPairs = append(tagPairs, secrets.TagPair{Key: key, Value: value})
			}
		}
	}
	
	// Sort by key, then by value
	sort.Slice(tagPairs, func(i, j int) bool {
		if tagPairs[i].Key != tagPairs[j].Key {
			return tagPairs[i].Key < tagPairs[j].Key
		}
		return tagPairs[i].Value < tagPairs[j].Value
	})
	
	return tagPairs
}

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

	err = utils.Render(w, r, views.StaticSecretDetailPage(details, string(hnd.config.App.Env)))
	if err != nil {
		hnd.log.Error("Failed to render static secret detail page: %v", err)
		status.AddToast(w, status.ErrorInternalServerError(err))
		return
	}
}

func (hnd *RouterHandler) ListStaticSecrets(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	cluster := r.URL.Query().Get("cluster")
	searchName := r.URL.Query().Get("search-name")

	secretsList, err := hnd.secretsService.ListSecrets(
		ctx,
		secrets.SecretTypeStaticSecret,
		cluster,
	)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
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

	return utils.Render(w, r, components.StaticSecretsTable(secretsList, string(hnd.config.App.Env), searchName, searchTagFilters, allTagPairs))
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

	additionalTags := parseAdditionalTags(r)

	req.Name = name
	req.Cluster = cluster
	req.Type = secrets.SecretTypeStaticSecret
	req.Value = value
	req.AdditionalTags = additionalTags

	resp, err := hnd.secretsService.CreateSecret(ctx, req)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	_, retryErr := utils.BackoffRetry(ctx, func() (*secrets.GetSecretResponse, error) {
		res, err := hnd.secretsService.GetSecret(ctx, resp.Name)
		if err != nil {
			return res, err
		}
		if res.Value != req.Value {
			return res, fmt.Errorf("secret value not yet set correctly")
		}
		return res, nil
	})
	if retryErr != nil {
		hnd.log.Warn("Secret created but not yet available after retries: %v", retryErr)
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
		return list, fmt.Errorf("secret not yet visible in list")
	})
	if err != nil {
		secretsList, err = hnd.secretsService.ListSecrets(ctx, req.Type, req.Cluster)
		if err != nil {
			status.AddToast(w, status.ErrorInternalServerError(err))
			return utils.Render(w, r, components.EmptyStaticSecretsTable())
		}
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Secret '%s' created successfully", resp.Name),
		StatusCode: http.StatusCreated,
	})
	allTagPairs := getAllTagPairs(secretsList)
	return utils.Render(w, r, components.StaticSecretsTable(secretsList, string(hnd.config.App.Env), "", make(map[string]string), allTagPairs))
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

	_, retryErr := utils.BackoffRetry(ctx, func() (*secrets.GetSecretResponse, error) {
		res, err := hnd.secretsService.GetSecret(ctx, resp.Name)
		if err != nil {
			return res, err
		}
		if res.Value != value {
			return res, fmt.Errorf("secret value not yet updated")
		}
		return res, nil
	})
	if retryErr != nil {
		hnd.log.Warn("Secret value updated but validation failed after retries: %v", retryErr)
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

	_, err := hnd.secretsService.DeleteSecret(ctx, secretName)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}
	// _, retryErr := utils.BackoffRetry(ctx, func() (*secrets.GetSecretResponse, error) {
	// 	res, err := hnd.secretsService.GetSecret(ctx, resp.Name)
	// 	if err != nil {
	// 		if strings.Contains(err.Error(), "ResourceNotFoundException") {
	// 			return nil, nil
	// 		}
	// 		return res, err
	// 	}
	// 	return res, fmt.Errorf("secret still exists")
	// })
	// if retryErr != nil {
	// 	hnd.log.Warn("Secret deletion validated but GetSecret check failed: %v", retryErr)
	// }

	cluster, secretType, err := secrets.ParseSecretName(secretName)
	if err != nil {
		status.AddToast(w, status.ErrorBadRequest(err))
		return utils.Render(w, r, components.EmptyStaticSecretsTable())
	}

	secretsList, err := utils.BackoffRetry(ctx, func() ([]secrets.Secret, error) {
		list, err := hnd.secretsService.ListSecrets(ctx, secretType, cluster)
		if err != nil {
			return list, err
		}
		for _, secret := range list {
			if secret.Name == secretName {
				return list, fmt.Errorf("secret still visible in list")
			}
		}
		return list, nil
	})
	if err != nil {
		secretsList, err = hnd.secretsService.ListSecrets(ctx, secretType, cluster)
		if err != nil {
			status.AddToast(w, status.ErrorInternalServerError(err))
			return utils.Render(w, r, components.EmptyStaticSecretsTable())
		}
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Secret '%s' is scheduled for deletion successfully", secretName),
		StatusCode: http.StatusOK,
	})
	allTagPairs := getAllTagPairs(secretsList)
	return utils.Render(w, r, components.StaticSecretsTable(secretsList, string(hnd.config.App.Env), "", make(map[string]string), allTagPairs))
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

	details, err := hnd.secretsService.GetSecretDetails(ctx, req.Name)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return nil
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Tag '%s' added successfully", key),
		StatusCode: http.StatusOK,
	})
	return utils.Render(w, r, views.StaticSecretDetailPage(details, string(hnd.config.App.Env)))
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

	details, err := hnd.secretsService.GetSecretDetails(ctx, req.Name)
	if err != nil {
		status.AddToast(w, status.ErrorInternalServerError(err))
		return nil
	}

	status.AddToast(w, status.Toast{
		Message:    fmt.Sprintf("Tag '%s' removed successfully", key),
		StatusCode: http.StatusOK,
	})
	return utils.Render(w, r, views.StaticSecretDetailPage(details, string(hnd.config.App.Env)))
}


