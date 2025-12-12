package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/iypetrov/lambdas/queue-event-to-slack/config"
	"github.com/iypetrov/lambdas/queue-event-to-slack/logger"
)

func Handler(ctx context.Context, event events.SQSEvent) (interface{}, error) {
	log := logger.Get(ctx)
	for _, msg := range event.Records {
		var s3Event events.S3Event
		if err := json.Unmarshal([]byte(msg.Body), &s3Event); err != nil {
			log.Error("Failed to parse S3 event from SQS message: %v", err)
			continue
		}

		for _, rec := range s3Event.Records {
			bucket := rec.S3.Bucket.Name
			object := rec.S3.Object.Key
			eventName := rec.EventName
			time := rec.EventTime
			log.Info(
				"S3 Event: %s occurred on object %s in bucket %s at %s",
				eventName,
				object,
				bucket,
				time,
			)
		}
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
