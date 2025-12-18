package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/config"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/logger"
)

type AuditTLSEvent struct {
	ID         string `dynamodbav:"id"`
	Action     string `dynamodbav:"action"`
	SecretName string `dynamodbav:"secret_name"`
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

func (s *Service) WriteAuditTLSEvent(ctx context.Context, secretName, cluster, action string, expireAt time.Time) error {
	event := AuditTLSEvent{
		ID:         uuid.New().String(),
		Action:     action,
		SecretName: secretName,
		Cluster:    cluster,
		ExpireAt:   expireAt.Unix(),
	}
	av, err := attributevalue.MarshalMap(event)
	if err != nil {
		return err
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String("audit-tls-events"),
		Item:      av,
	})
	if err != nil {
		return err
	}
	return nil
}
