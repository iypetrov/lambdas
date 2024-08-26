package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/iypetrov/lambdas/placeholder/pkg/logger"
	"go.uber.org/zap"
)

var ctx context.Context

func init() {
	ctx = context.Background()

	log, _ := zap.NewProduction()
	ctx = logger.Inject(ctx, log)
}

func Handler(ctx context.Context, event interface{}) (interface{}, error) {
	log := logger.GetLoggerFromContext(ctx)
	log.Info("received event", zap.Any("event", event))
	return event, nil
}

func main() {
	lambda.StartWithOptions(Handler, lambda.WithContext(ctx))
}
