package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/config"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/logger"
)

type AuditEvent struct {
	ID         string `dynamodbav:"id"`
	Action     string `dynamodbav:"action"`
	SecretName string `dynamodbav:"secret_name"`
	SecretType string `dynamodbav:"secret_type"`
	Cluster	   string `dynamodbav:"cluster"`
	ExpireAt   int64  `dynamodbav:"expire_at"`
}

type Service struct {
	client *dynamodb.Client
	log    logger.Logger
}

func NewService(ctx context.Context, cfg config.Config, log logger.Logger) *Service {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Error("Failed to load AWS config: %v", err)
		return nil
	}

	return &Service{
		client: dynamodb.NewFromConfig(awsCfg),
		log:    log,
	}
}

func (s *Service) WriteAuditEvent(ctx context.Context, secretName, secretType, cluster, action string, expireAt time.Time) (string, error) {
	id := uuid.New().String()
	event := AuditEvent{
		ID:         id,
		Action:     action,
		SecretName: secretName,
		SecretType: secretType,
		Cluster:    cluster,
		ExpireAt:   expireAt.Unix(),
	}
	av, err := attributevalue.MarshalMap(event)
	if err != nil {
		return "", err
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("audit-tls-events"),
		Item:      av,
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

func(s *Service) AddArn(ctx context.Context, id, arn string) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String("audit-tls-events"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression: aws.String("SET arn = :arn"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":arn": &types.AttributeValueMemberS{Value: arn},
		},
	})
	return err
}
