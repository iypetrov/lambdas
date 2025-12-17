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
	log.Info("Received CloudWatch Event")
	log.Info("version: %s", event.Version)
	log.Info("id: %s", event.ID)
	log.Info("detail-type: %s", event.DetailType)
	log.Info("source: %s", event.Source)
	log.Info("account: %s", event.AccountID)
	log.Info("time: %s", event.Time.String())
	log.Info("region: %s", event.Region)
	log.Info("resources: %v", event.Resources)

	var detail SecretsManagerEventDetail
	if err := json.Unmarshal(event.Detail, &detail); err != nil {
		log.Error("failed to unmarshal detail: %v", err)
		return nil, err
	}

	log.Info("SecretsManager Event Detail:")
	log.Info("EventName: %s", detail.EventName)
	log.Info("AWSRegion: %s", detail.AWSRegion)
	log.Info("EventTime: %s", detail.EventTime)
	log.Info("RequestParameters: %v", detail.RequestParameters)
	log.Info("ResponseElements: %v", detail.ResponseElements)
	log.Info("Resources: %v", detail.Resources)

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
