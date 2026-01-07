package secretsmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	awssmtype "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/config"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/logger"
)

type SecretType string

const (
	SecretTypeStaticSecret  SecretType = "Static Secret"
	SecretTypeTLSCertificate SecretType = "TLS Certificate"
)

const (
	TagCategory   = "Category"
	TagCluster    = "Cluster"
	TagType       = "Type"
	TagManagedBy  = "ManagedBy"
)

var Tags = map[string]bool{
	TagCategory:  true,
	TagCluster:   true,
	TagType:      true,
	TagManagedBy: true,
}

func IsProtectedTag(tagKey string) bool {
	return Tags[tagKey]
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

type TLSCertificateData struct {
	Crt string `json:"crt"`
	Key string `json:"key"`
}

type AddTagRequest struct {
	Name  string `json:"name"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Service struct {
	client *secretsmanager.Client
	log    logger.Logger
}

func NewService(ctx context.Context, cfg config.Config, log logger.Logger) *Service {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Error("Failed to load AWS config: %v", err)
		return nil
	}

	return &Service{
		client: secretsmanager.NewFromConfig(awsCfg),
		log:    log,
	}
}

func (s *Service) GetSecretDetails(ctx context.Context, secretName string) (*GetSecretDetailsResponse, error) {
	describeReq := &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(secretName),
	}

	result, err := s.client.DescribeSecret(ctx, describeReq)
	if err != nil {
		s.log.Error("Error describing secret %s: %v", secretName, err)
		return nil, fmt.Errorf("%s", err.Error())
	}

	tags := convertToTags(result.Tags)
	cluster, secretType, err := ParseSecretName(*result.Name)
	if err != nil {
		s.log.Error("Error parsing secret name %s: %v", *result.Name, err)
		return nil, fmt.Errorf("error parsing secret name: %s", err.Error())
	}

	var createdDate string
	if result.CreatedDate != nil {
		createdDate = result.CreatedDate.Format(time.RFC3339)
	}

	var lastChangedDate string
	if result.LastChangedDate != nil {
		lastChangedDate = result.LastChangedDate.Format(time.RFC3339)
	}

	var lastRotatedDate string
	if result.LastRotatedDate != nil {
		lastRotatedDate = result.LastRotatedDate.Format(time.RFC3339)
	}

	var description string
	if result.Description != nil {
		description = *result.Description
	}

	s.log.Info("Retrieved secret details for %s", secretName)
	return &GetSecretDetailsResponse{
		Name:            *result.Name,
		ARN:             *result.ARN,
		Type:            secretType,
		Cluster:         cluster,
		Tags:            tags,
		CreatedDate:     createdDate,
		LastChangedDate: lastChangedDate,
		LastRotatedDate: lastRotatedDate,
		Description:     description,
	}, nil
}

func (s *Service) GetSecretTLS(ctx context.Context, secretName string) (TLSCertificateData, error) {
	req := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := s.client.GetSecretValue(ctx, req)
	if err != nil {
		s.log.Error("Error retrieving secret %s: %v", secretName, err)
		return TLSCertificateData{}, fmt.Errorf("%s", err.Error())
	}

	var tlsData TLSCertificateData
	err = json.Unmarshal([]byte(*result.SecretString), &tlsData)
	if err != nil {
		s.log.Error("Error unmarshaling secret %s: %v", secretName, err)
		return TLSCertificateData{}, fmt.Errorf("error unmarshaling secret: %s", err.Error())
	}

	s.log.Info("Retrieved TLS data for secret %s", secretName)
	return tlsData, nil
}

func (s *Service) AddTag(ctx context.Context, req AddTagRequest) error {
	tagReq := &secretsmanager.TagResourceInput{
		SecretId: aws.String(req.Name),
		Tags: []awssmtype.Tag{
			{
				Key:   aws.String(req.Key),
				Value: aws.String(req.Value),
			},
		},
	}
	_, err := s.client.TagResource(ctx, tagReq)
	if err != nil {
		s.log.Error("Error adding tag to secret %s: %v", req.Name, err)
		return fmt.Errorf("%s", err.Error())
	}
	return nil
}
