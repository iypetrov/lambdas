package secrets

type SecretType string

const (
	SecretTypeStaticSecret  SecretType = "Static Secret"
	SecretTypeTLSCertificate SecretType = "TLS Certificate"
)

// Protected tag keys that cannot be modified or deleted
const (
	ProtectedTagCategory   = "Category"
	ProtectedTagCluster    = "Cluster"
	ProtectedTagType       = "Type"
	ProtectedTagManagedBy  = "ManagedBy"
)

// ProtectedTags is a set of protected tag keys
var ProtectedTags = map[string]bool{
	ProtectedTagCategory:  true,
	ProtectedTagCluster:   true,
	ProtectedTagType:      true,
	ProtectedTagManagedBy: true,
}

// IsProtectedTag checks if a tag key is protected
func IsProtectedTag(tagKey string) bool {
	return ProtectedTags[tagKey]
}

// TagMap is an ordered map of tags (key-value pairs)
type TagMap []TagPair

// TagPair represents a single tag key-value pair
type TagPair struct {
	Key   string
	Value string
}

// ToMap converts TagMap to a regular map for backward compatibility
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

type GetSecretDetailsResponse struct {
	Name         string            `json:"name"`
	ARN          string            `json:"arn"`
	Type         SecretType        `json:"type"`
	Cluster      string            `json:"cluster"`
	Tags         TagMap            `json:"tags"`
	VersionID    string            `json:"version_id,omitempty"`
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

// TLSCertificateData represents the structure for TLS certificate data stored as JSON
type TLSCertificateData struct {
	Crt string `json:"crt"`
	Key string `json:"key"`
}

