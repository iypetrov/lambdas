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
	eventTime := detail.EventTime
	if len(detail.ResponseElements) == 0 {
		log.Info("No response elements found for event %s at %s", eventName, eventTime)
		return detail, nil
	}
	if len(detail.ResponseElements) >= 1 {
		log.Warn("Response elements found for event %s at %s: %v", eventName, eventTime, detail.ResponseElements)
	}

	arn, ok := detail.ResponseElements["arn"].(string); 
	if !ok {
		log.Error("No ARN found in response elements")
		return nil, fmt.Errorf("no ARN found in response elements")
	}

	log.Info("EventName: %s", eventName)
	log.Info("EventTime: %s", eventTime)
	log.Info("ResponseElement ARN: %v", arn)

	arnParts := strings.Split(arn, ":")
    secretNameWithSuffix := arnParts[len(arnParts)-1]
	log.Info("Secret Name with Suffix: %s", secretNameWithSuffix)
	secretNameWithSuffixParts := strings.Split(secretNameWithSuffix, "-")
	secretName := strings.Join(secretNameWithSuffixParts[:len(secretNameWithSuffixParts)-1], "-")
	log.Info("Secret Name: %s", secretName)

	secretDetail, err := secretsMangerService.GetSecretDetails(ctx, secretName)
	if err != nil {
		log.Error("Failed to get secret details for %s: %v", secretName, err)
		return nil, err
	}

	for _, tag := range secretDetail.Tags {
		if tag.Key == secretsmanager.TagType && tag.Value != string(secretsmanager.SecretTypeTLSCertificate) {
			log.Error("Secret %s is not of type TLS Certificate, skipping", secretName)
			return detail, fmt.Errorf("secret %s is not of type TLS Certificate, skipping", secretName)
		}

		if tag.Key == secretsmanager.TagCategory && tag.Value != "Restricted" {
			log.Error("Secret %s is not in Restricted category, skipping", secretName)
			return detail, fmt.Errorf("secret %s is not in Restricted category, skipping", secretName)
		}
	}

	log.Info("Secret %s passed validation checks, proceeding with ACM import", secretName)

	t, err := time.Parse(time.RFC3339, "2025-12-17T15:00:55Z")
	if err != nil {
		log.Error("Failed to parse time: %v", err)
		return nil, err
	}

	err = dynamodbService.WriteAuditTLSEvent(ctx, secretName, eventName, t)
	if err != nil { 
		log.Error("Failed to write audit TLS event for secret %s: %v", secretName, err)
		return nil, err
	}

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
