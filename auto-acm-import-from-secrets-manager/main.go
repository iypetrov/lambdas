package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/config"
	"github.com/iypetrov/lambdas/auto-acm-import-from-secrets-manager/logger"
)

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
	log.Info("detail: %v", event.Detail)
	return event, nil
}

func main() {
	ctx := context.Background()
	cfg := config.New()
	log := logger.New(cfg)
	ctx = log.Inject(ctx)
	ctx = config.Inject(ctx, *cfg)

	lambda.StartWithOptions(Handler, lambda.WithContext(ctx))
}
