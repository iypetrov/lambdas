package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/config"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/dynamodb"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/logger"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/secretsmanager"
)

type SecretsManagerEventDetail struct {
	EventVersion      string                   `json:"eventVersion"`
	UserIdentity      map[string]interface{}   `json:"userIdentity"`
	EventTime         string                   `json:"eventTime"`
	EventName         string                   `json:"eventName"`
	AWSRegion         string                   `json:"awsRegion"`
	SourceIPAddress   string                   `json:"sourceIPAddress"`
	RequestParameters map[string]interface{}   `json:"requestParameters"`
	ResponseElements  map[string]interface{}   `json:"responseElements"`
	EventID           string                   `json:"eventID"`
	ReadOnly          bool                     `json:"readOnly"`
	Resources         []map[string]interface{} `json:"resources"`
}

func Handler(ctx context.Context, event events.CloudWatchEvent) (interface{}, error) {
	log := logger.Get(ctx)
	cfg := config.Get(ctx)
	secretsMangerService := secretsmanager.NewService(ctx, cfg, log)
	dynamodbService := dynamodb.NewService(ctx, cfg, log)

	var detail SecretsManagerEventDetail
	if err := json.Unmarshal(event.Detail, &detail); err != nil {
		log.Error("failed to unmarshal detail: %v", err)
		return nil, err
	}

	eventName := detail.EventName
	if len(detail.ResponseElements) == 0 {
		log.Info("No response elements found for event %s", eventName)
		return detail, nil
	}
	if len(detail.ResponseElements) > 1 {
		log.Warn("Response elements found for event %s: %v", eventName, detail.ResponseElements)
	}

	arn, ok := detail.ResponseElements["arn"].(string); 
	if !ok {
		log.Error("No ARN found in response elements")
		return nil, fmt.Errorf("no ARN found in response elements")
	}

	arnParts := strings.Split(arn, ":")
    secretNameWithSuffix := arnParts[len(arnParts)-1]
	secretNameWithSuffixParts := strings.Split(secretNameWithSuffix, "-")
	secretName := strings.Join(secretNameWithSuffixParts[:len(secretNameWithSuffixParts)-1], "-")

	cluster := strings.Split(secretName, ".")[1]
	secretType := strings.Split(secretName, ".")[2]

	secretDetail, err := secretsMangerService.GetSecretDetails(ctx, secretName)
	if err != nil {
		log.Error("Failed to get secret details for %s: %v", secretName, err)
		return nil, err
	}

	for _, tag := range secretDetail.Tags {
		if tag.Key == secretsmanager.TagCategory && tag.Value != "Restricted" {
			log.Error("Secret %s is not in Restricted category, skipping", secretName)
			return detail, fmt.Errorf("secret %s is not in Restricted category, skipping", secretName)
		}
	}
	log.Info("Secret %s passed validation check for restricted secret", secretName)

	err = dynamodbService.WriteAuditTLSEvent(
		ctx, 
		secretName, 
		secretType,
		cluster,
		eventName, 
		time.Now().UTC().Add(168 * time.Hour),
	)
	if err != nil { 
		log.Error("Failed to write audit TLS event for secret %s: %v", secretName, err)
		return nil, err
	}
	log.Info("Secret %s was inserted in the audit-tls-events DynamoDB table", secretName)

	for _, tag := range secretDetail.Tags {
		if tag.Key == secretsmanager.TagType && tag.Value != string(secretsmanager.SecretTypeTLSCertificate) {
			log.Error("Secret %s is not of type TLS Certificate, skipping", secretName)
			return detail, fmt.Errorf("secret %s is not of type TLS Certificate, skipping", secretName)
		}
	}
	log.Info("Secret %s passed validation check for TLS Certificate type", secretName)

	return detail, nil
}

func main() {
	ctx := context.Background()
	cfg := config.New()
	log := logger.New(cfg)
	ctx = log.Inject(ctx)
	ctx = config.Inject(ctx, *cfg)

	lambda.StartWithOptions(Handler, lambda.WithContext(ctx))
}
