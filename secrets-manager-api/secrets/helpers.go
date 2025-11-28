package secrets

import (
	"fmt"
	"strings"
)

// ParseSecretName extracts cluster and type from a secret name
// Format: RESTRICTED.<cluster>.<TYPE>.<name>
// Returns: cluster, secretType, error
func ParseSecretName(secretName string) (string, SecretType, error) {
	parts := strings.Split(secretName, ".")
	if len(parts) < 4 || parts[0] != "RESTRICTED" {
		return "", "", fmt.Errorf("invalid secret name format")
	}

	cluster := parts[1]
	typePart := parts[2]

	var secretType SecretType
	switch typePart {
	case "STATIC_SECRET":
		secretType = SecretTypeStaticSecret
	case "TLS_CERTIFICATE":
		secretType = SecretTypeTLSCertificate
	default:
		return "", "", fmt.Errorf("unknown secret type: %s", typePart)
	}

	return cluster, secretType, nil
}
