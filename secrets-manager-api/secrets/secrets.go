package secrets

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/smithy-go"
	"github.com/iypetrov/lambdas/secrets-manager-api/config"
	"github.com/iypetrov/lambdas/secrets-manager-api/logger"
)

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

func (s *Service) buildSecretName(secretType SecretType, cluster, name string) string {
	var typePart string
	if secretType == SecretTypeStaticSecret {
		typePart = "STATIC_SECRET"
	} else {
		typePart = "TLS_CERTIFICATE"
	}
	return fmt.Sprintf("RESTRICTED.%s.%s.%s", cluster, typePart, name)
}

func (s *Service) buildTags(secretType SecretType, cluster string, additionalTags map[string]string) []types.Tag {
	tags := []types.Tag{
		{
			Key:   aws.String("Category"),
			Value: aws.String("Restricted"),
		},
		{
			Key:   aws.String("Cluster"),
			Value: aws.String(cluster),
		},
		{
			Key:   aws.String("Type"),
			Value: aws.String(string(secretType)),
		},
		{
			Key:   aws.String("ManagedBy"),
			Value: aws.String("secrets-manager-api"),
		},
	}

	for key, value := range additionalTags {
		tags = append(tags, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}

	return tags
}

func (s *Service) CreateSecret(ctx context.Context, req CreateSecretRequest) (*CreateSecretResponse, error) {
	secretName := s.buildSecretName(req.Type, req.Cluster, req.Name)
	tags := s.buildTags(req.Type, req.Cluster, req.AdditionalTags)

	createReq := &secretsmanager.CreateSecretInput{
		Name:         aws.String(secretName),
		SecretString: aws.String(req.Value),
		Description:  aws.String("Secret managed by secrets-manager-api"),
		Tags:         tags,
	}

	result, err := s.client.CreateSecret(ctx, createReq)
	if err != nil {
		var ae smithy.APIError
		if strings.Contains(err.Error(), "ResourceExistsException") {
			s.log.Error("Secret already exists: %s", secretName)
			return nil, fmt.Errorf("secret already exists: %s", secretName)
		}
		if strings.Contains(err.Error(), "InvalidRequestException") {
			s.log.Error("Invalid request creating secret %s: %v", secretName, err)
			return nil, fmt.Errorf("invalid request creating secret: %s", secretName)
		}
		if errors.As(err, &ae) {
			s.log.Error("AWS error creating secret %s: %v", secretName, err)
			return nil, fmt.Errorf("AWS error creating secret: %s", err.Error())
		}
		s.log.Error("Unexpected error creating secret %s: %v", secretName, err)
		return nil, fmt.Errorf("unexpected error creating secret: %s", err.Error())
	}

	s.log.Info("Created secret %s", *result.ARN)

	return &CreateSecretResponse{
		Name:      *result.Name,
		ARN:       *result.ARN,
		VersionID: *result.VersionId,
	}, nil
}

func (s *Service) GetSecret(ctx context.Context, secretName string) (*GetSecretResponse, error) {
	getReq := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := s.client.GetSecretValue(ctx, getReq)
	if err != nil {
		var ae smithy.APIError
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			s.log.Error("Secret not found: %s", secretName)
			return nil, fmt.Errorf("secret not found: %s", secretName)
		}
		if errors.As(err, &ae) {
			s.log.Error("AWS error getting secret %s: %v", secretName, err)
			return nil, fmt.Errorf("AWS error getting secret: %s", err.Error())
		}
		s.log.Error("Unexpected error getting secret %s: %v", secretName, err)
		return nil, fmt.Errorf("unexpected error getting secret: %s", err.Error())
	}

	s.log.Info("Retrieved secret %s", secretName)

	return &GetSecretResponse{
		Value: *result.SecretString,
	}, nil
}

func (s *Service) UpdateSecret(ctx context.Context, req UpdateSecretRequest) (*UpdateSecretResponse, error) {
	putReq := &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(req.Name),
		SecretString: aws.String(req.Value),
	}

	result, err := s.client.PutSecretValue(ctx, putReq)
	if err != nil {
		var ae smithy.APIError
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			s.log.Error("Secret not found for update: %s", req.Name)
			return nil, fmt.Errorf("secret not found: %s", req.Name)
		}
		if errors.As(err, &ae) {
			s.log.Error("AWS error updating secret %s: %v", req.Name, err)
			return nil, fmt.Errorf("AWS error updating secret: %s", err.Error())
		}
		s.log.Error("Unexpected error updating secret %s: %v", req.Name, err)
		return nil, fmt.Errorf("unexpected error updating secret: %s", err.Error())
	}

	s.log.Info("Updated secret %s", *result.ARN)

	return &UpdateSecretResponse{
		Name:      *result.Name,
		ARN:       *result.ARN,
		VersionID: *result.VersionId,
	}, nil
}

func (s *Service) DeleteSecret(ctx context.Context, secretName string) (*DeleteSecretResponse, error) {
	// Check if secret is already scheduled for deletion
	describeReq := &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(secretName),
	}

	describe, err := s.client.DescribeSecret(ctx, describeReq)
	if err != nil {
		if !strings.Contains(err.Error(), "ResourceNotFoundException") {
			s.log.Warn("DescribeSecret failed for %s. Proceeding to delete.", secretName)
		}
	} else if describe.DeletedDate != nil {
		return nil, fmt.Errorf("secret is already scheduled for deletion at: %s", describe.DeletedDate.Format(time.RFC3339))
	}

	deleteReq := &secretsmanager.DeleteSecretInput{
		SecretId:                aws.String(secretName),
		ForceDeleteWithoutRecovery: aws.Bool(true),
	}

	result, err := s.client.DeleteSecret(ctx, deleteReq)
	if err != nil {
		var ae smithy.APIError
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			s.log.Error("Secret not found for deletion: %s", secretName)
			return nil, fmt.Errorf("secret not found: %s", secretName)
		}
		if errors.As(err, &ae) {
			s.log.Error("AWS error deleting secret %s: %v", secretName, err)
			return nil, fmt.Errorf("AWS error deleting secret: %s", err.Error())
		}
		s.log.Error("Unexpected error deleting secret %s: %v", secretName, err)
		return nil, fmt.Errorf("unexpected error deleting secret: %s", err.Error())
	}

	var deletionDate string
	if result.DeletionDate != nil {
		deletionDate = result.DeletionDate.Format(time.RFC3339)
	}

	s.log.Info("Deleted secret %s", *result.ARN)

	return &DeleteSecretResponse{
		Name:         *result.Name,
		ARN:          *result.ARN,
		DeletionDate: deletionDate,
	}, nil
}

func (s *Service) ListSecrets(ctx context.Context, secretType SecretType, cluster string) ([]Secret, error) {
	var secrets []Secret
	var nextToken *string

	for {
		listReq := &secretsmanager.ListSecretsInput{
			NextToken: nextToken,
		}

		result, err := s.client.ListSecrets(ctx, listReq)
		if err != nil {
			s.log.Error("AWS error listing secrets: %v", err)
			return nil, fmt.Errorf("AWS error listing secrets: %s", err.Error())
		}

		for _, secret := range result.SecretList {
			// Filter by tags
			secretTypeTag := ""
			secretCluster := ""
			tags := make(map[string]string)

			for _, tag := range secret.Tags {
				if tag.Key != nil && tag.Value != nil {
					tags[*tag.Key] = *tag.Value
					if *tag.Key == "Type" {
						secretTypeTag = *tag.Value
					}
					if *tag.Key == "Cluster" {
						secretCluster = *tag.Value
					}
				}
			}

			// Only include secrets that match the requested type and cluster
			if secretTypeTag == string(secretType) && (cluster == "" || secretCluster == cluster) {
				var createdDate string
				if secret.CreatedDate != nil {
					createdDate = secret.CreatedDate.Format(time.RFC3339)
				}

				secrets = append(secrets, Secret{
					Name:        *secret.Name,
					ARN:         *secret.ARN,
					Type:        secretType,
					Cluster:     secretCluster,
					Tags:        tags,
					CreatedDate: createdDate,
				})
			}
		}

		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}

	return secrets, nil
}

func (s *Service) GetStatistics(ctx context.Context, cluster string) (*Statistics, error) {
	staticSecrets, err := s.ListSecrets(ctx, SecretTypeStaticSecret, cluster)
	if err != nil {
		return nil, err
	}

	tlsCertificates, err := s.ListSecrets(ctx, SecretTypeTLSCertificate, cluster)
	if err != nil {
		return nil, err
	}

	return &Statistics{
		StaticSecretsCount:  len(staticSecrets),
		TLSCertificatesCount: len(tlsCertificates),
	}, nil
}
