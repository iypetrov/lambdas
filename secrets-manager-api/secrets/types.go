package secrets

type SecretType string

const (
	SecretTypeStaticSecret  SecretType = "Static Secret"
	SecretTypeTLSCertificate SecretType = "TLS Certificate"
)

type Secret struct {
	Name        string            `json:"name"`
	ARN         string            `json:"arn"`
	Type        SecretType        `json:"type"`
	Cluster     string            `json:"cluster"`
	Tags        map[string]string `json:"tags"`
	VersionID   string            `json:"version_id,omitempty"`
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
	VersionID string `json:"version_id"`
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
	VersionID string `json:"version_id"`
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

