package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/config"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/logger"
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

	if arn, ok := detail.ResponseElements["arn"]; ok {
		log.Info("ARN found: %v", arn)
	} else {
		log.Info("No ARN found in response elements")
	}

	log.Info("EventName: %s", eventName)
	log.Info("EventTime: %s", eventTime)
	log.Info("ResponseElement ARN: %v", detail.ResponseElements["arn"])

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
