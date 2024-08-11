package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/iypetrov/lambdas/placeholder/pkg/logger"
	"go.uber.org/zap"
)

var ctx context.Context

func init() {
	log, err := zap.NewProduction()
	if err != nil {
		log.Fatal("couldn't set up logger", zap.Error(err))
	}
	defer func(log *zap.Logger) {
		err := log.Sync()
		if err != nil {
			log.Fatal("couldn't sync logger", zap.Error(err))
		}
	}(log)

	ctx = context.Background()
	ctx = logger.Inject(ctx, log)
}

func Handler(ctx context.Context, event interface{}) error {
	log := logger.GetLoggerFromContext(ctx)
	log.Info("received event lambda", zap.Any("event", event))
	return nil
}

func main() {
	lambda.StartWithOptions(Handler, lambda.WithContext(ctx))
}
