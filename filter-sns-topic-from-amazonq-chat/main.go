package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/ip812/filter-sns-topic-from-amazonq-chat/config"
	"github.com/ip812/filter-sns-topic-from-amazonq-chat/logger"
)

func Handler(ctx context.Context, event interface{}) (interface{}, error) {
	log := logger.Get(ctx)
	log.Info("Hello, received event is: %v", event)
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
