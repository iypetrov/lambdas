package secrets

import (
	awssmtype "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

type SecretType string

const (
	SecretTypeStaticSecret  SecretType = "Static Secret"
	SecretTypeTLSCertificate SecretType = "TLS Certificate"
)

const (
	ProtectedTagCategory   = "Category"
	ProtectedTagCluster    = "Cluster"
	ProtectedTagType       = "Type"
	ProtectedTagManagedBy  = "ManagedBy"
)

var ProtectedTags = map[string]bool{
	ProtectedTagCategory:  true,
	ProtectedTagCluster:   true,
	ProtectedTagType:      true,
	ProtectedTagManagedBy: true,
}

func IsProtectedTag(tagKey string) bool {
	return ProtectedTags[tagKey]
}

type TagMap []TagPair

type TagPair struct {
	Key   string
	Value string
}

func convertToTags(awstp []awssmtype.Tag) TagMap {
	var tm TagMap
	for _, t := range awstp {
		tm = append(tm, TagPair{
			Key:   *t.Key,
			Value: *t.Value,
		})
	}
	return tm
}

func (tm TagMap) ToMap() map[string]string {
	result := make(map[string]string, len(tm))
	for _, pair := range tm {
		result[pair.Key] = pair.Value
	}
	return result
}

type Secret struct {
	Name        string            `json:"name"`
	ARN         string            `json:"arn"`
	Type        SecretType        `json:"type"`
	Cluster     string            `json:"cluster"`
	Tags        map[string]string `json:"tags"`
	CreatedDate string            `json:"created_date,omitempty"`
}

type CreateSecretRequest struct {
	Name        string            `json:"name"`
	Value       string            `json:"value"`
	Type        SecretType        `json:"type"`
	Cluster     string            `json:"cluster"`
	AdditionalTags map[string]string `json:"additional_tags,omitempty"`
}

type CreateSecretResponse struct {
	Name      string `json:"name"`
	ARN       string `json:"arn"`
}

type GetSecretResponse struct {
	Value string `json:"value"`
}

type UpdateSecretRequest struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type UpdateSecretResponse struct {
	Name      string `json:"name"`
	ARN       string `json:"arn"`
}

type DeleteSecretResponse struct {
	Name        string `json:"name"`
	ARN         string `json:"arn"`
	DeletionDate string `json:"deletion_date,omitempty"`
}

type Statistics struct {
	StaticSecretsCount  int `json:"static_secrets_count"`
	TLSCertificatesCount int `json:"tls_certificates_count"`
}

type GetSecretDetailsResponse struct {
	Name         string            `json:"name"`
	ARN          string            `json:"arn"`
	Type         SecretType        `json:"type"`
	Cluster      string            `json:"cluster"`
	Tags         TagMap            `json:"tags"`
	CreatedDate  string            `json:"created_date,omitempty"`
	LastChangedDate string         `json:"last_changed_date,omitempty"`
	LastRotatedDate  string        `json:"last_rotated_date,omitempty"`
	Description  string            `json:"description,omitempty"`
}

type AddTagRequest struct {
	Name  string `json:"name"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RemoveTagRequest struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type UpdateTagsResponse struct {
	Name string `json:"name"`
	ARN  string `json:"arn"`
}

type TLSCertificateData struct {
	Crt string `json:"crt"`
	Key string `json:"key"`
}
