package secrets

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
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
		s.log.Error("Error creating secret %s: %v", secretName, err)
		return nil, fmt.Errorf("%s", err.Error())
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
		s.log.Error("Error retrieving secret %s: %v", secretName, err)
		return nil, fmt.Errorf("%s", err.Error())
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

func (s *Service) GetSecretDetails(ctx context.Context, secretName string) (*GetSecretDetailsResponse, error) {
	describeReq := &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(secretName),
	}

	result, err := s.client.DescribeSecret(ctx, describeReq)
	if err != nil {
		s.log.Error("Error describing secret %s: %v", secretName, err)
		return nil, fmt.Errorf("%s", err.Error())
	}

	tagsMap := make(map[string]string)
	var secretType SecretType
	var cluster string

	for _, tag := range result.Tags {
		if tag.Key != nil && tag.Value != nil {
			tagsMap[*tag.Key] = *tag.Value
			if *tag.Key == "Type" {
				if *tag.Value == string(SecretTypeStaticSecret) {
					secretType = SecretTypeStaticSecret
				} else if *tag.Value == string(SecretTypeTLSCertificate) {
					secretType = SecretTypeTLSCertificate
				}
			}
			if *tag.Key == "Cluster" {
				cluster = *tag.Value
			}
		}
	}

	// Sort tag keys alphabetically and create ordered TagMap
	sortedTagKeys := make([]string, 0, len(tagsMap))
	for key := range tagsMap {
		sortedTagKeys = append(sortedTagKeys, key)
	}
	sort.Strings(sortedTagKeys)

	tags := make(TagMap, 0, len(sortedTagKeys))
	for _, key := range sortedTagKeys {
		tags = append(tags, TagPair{
			Key:   key,
			Value: tagsMap[key],
		})
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

	var versionID string
	if result.VersionIdsToStages != nil && len(result.VersionIdsToStages) > 0 {
		// Get the AWSCURRENT version ID if available, otherwise get the first one
		for vID, stages := range result.VersionIdsToStages {
			for _, stage := range stages {
				if stage == "AWSCURRENT" {
					versionID = vID
					break
				}
			}
			if versionID != "" {
				break
			}
		}
		// If no AWSCURRENT found, get the first version ID
		if versionID == "" {
			for vID := range result.VersionIdsToStages {
				versionID = vID
				break
			}
		}
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
		VersionID:       versionID,
		CreatedDate:     createdDate,
		LastChangedDate: lastChangedDate,
		LastRotatedDate: lastRotatedDate,
		Description:     description,
	}, nil
}

func (s *Service) AddTag(ctx context.Context, req AddTagRequest) (*UpdateTagsResponse, error) {
	tagReq := &secretsmanager.TagResourceInput{
		SecretId: aws.String(req.Name),
		Tags: []types.Tag{
			{
				Key:   aws.String(req.Key),
				Value: aws.String(req.Value),
			},
		},
	}

	_, err := s.client.TagResource(ctx, tagReq)
	if err != nil {
		s.log.Error("Error adding tag to secret %s: %v", req.Name, err)
		return nil, fmt.Errorf("%s", err.Error())
	}

	// Get the ARN for the response
	describeReq := &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(req.Name),
	}
	describe, err := s.client.DescribeSecret(ctx, describeReq)
	if err != nil {
		s.log.Error("Error describing secret %s: %v", req.Name, err)
		return nil, fmt.Errorf("%s", err.Error())
	}

	s.log.Info("Added tag %s=%s to secret %s", req.Key, req.Value, req.Name)
	return &UpdateTagsResponse{
		Name: req.Name,
		ARN:  *describe.ARN,
	}, nil
}

func (s *Service) RemoveTag(ctx context.Context, req RemoveTagRequest) (*UpdateTagsResponse, error) {
	untagReq := &secretsmanager.UntagResourceInput{
		SecretId: aws.String(req.Name),
		TagKeys:  []string{req.Key},
	}

	_, err := s.client.UntagResource(ctx, untagReq)
	if err != nil {
		s.log.Error("Error removing tag from secret %s: %v", req.Name, err)
		return nil, fmt.Errorf("%s", err.Error())
	}

	// Get the ARN for the response
	describeReq := &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(req.Name),
	}
	describe, err := s.client.DescribeSecret(ctx, describeReq)
	if err != nil {
		s.log.Error("Error describing secret %s: %v", req.Name, err)
		return nil, fmt.Errorf("%s", err.Error())
	}

	s.log.Info("Removed tag %s from secret %s", req.Key, req.Name)
	return &UpdateTagsResponse{
		Name: req.Name,
		ARN:  *describe.ARN,
	}, nil
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

