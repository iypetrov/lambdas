package metadata

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/iypetrov/lambdas/secrets-manager-api/config"
	"github.com/iypetrov/lambdas/secrets-manager-api/logger"
)

type AuditTLSEvent struct {
	ID         string `dynamodbav:"id"`
	Action     string `dynamodbav:"action"`
	SecretName string `dynamodbav:"secret_name"`
	SecretType string `dynamodbav:"secret_type"`
	Cluster    string `dynamodbav:"cluster"`
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

func (s *Service) GetAuditEvent(ctx context.Context, cluster string) ([]AuditTLSEvent, error) {
	var events []AuditTLSEvent

	paginator := dynamodb.NewScanPaginator(s.client, &dynamodb.ScanInput{
		TableName: aws.String("audit-tls-events"),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			s.log.Error("Failed to get next page: %v", err)
		}

		var pageEvents []AuditTLSEvent
		err = attributevalue.UnmarshalListOfMaps(page.Items, &pageEvents)
		if err != nil {
			s.log.Error("Failed to unmarshal page items: %v", err)
		}

		events = append(events, pageEvents...)
	}

	var filteredEvents []AuditTLSEvent
	for _, event := range events {
		if event.Cluster == cluster {
			filteredEvents = append(filteredEvents, event)
		}
	}

	return filteredEvents, nil
}
