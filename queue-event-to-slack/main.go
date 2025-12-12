package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/iypetrov/lambdas/queue-event-to-slack/config"
	"github.com/iypetrov/lambdas/queue-event-to-slack/logger"
)

func Handler(ctx context.Context, event events.SQSEvent) (interface{}, error) {
	log := logger.Get(ctx)
	log.Info("Hello, received event is: %v", event)
	for _, message := range event.Records {
		log.Info("Message ID: %s", message.MessageId)
		log.Info("Body: %s", message.Body)
		log.Info("Attributes: %#v", message.Attributes)
		log.Info("MessageAttributes: %#v", message.MessageAttributes)
		log.Info("----")
	}
	return event, nil
}

func main() {
	ctx := context.Background()
	cfg := config.New()
	log := logger.New(cfg)
	ctx = log.Inject(ctx)
	ctx = config.Inject(ctx, cfg)

	lambda.StartWithOptions(Handler, lambda.WithContext(ctx))
}
