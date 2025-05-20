package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/iypetrov/lambdas/filter-sns-topic-from-amazonq-chat/config"
	"github.com/iypetrov/lambdas/filter-sns-topic-from-amazonq-chat/logger"
)

func Handler(ctx context.Context, event events.SNSEvent) (events.SNSEvent, error) {
	log := logger.Get(ctx)
	cfg := config.Get(ctx)

	awsCfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Error("Failed to load AWS config: %s", err.Error())
		return events.SNSEvent{Records: []events.SNSEventRecord{}}, err
	}

	log.Info("Hello, received event is: %v", event)
	snsClient := sns.NewFromConfig(awsCfg)
	for _, record := range event.Records {
		msg := record.SNS.Message
		subject := record.SNS.Subject

		log.Info("Forwarding message from subject '%s'", subject)

		_, err := snsClient.Publish(ctx, &sns.PublishInput{
			Message:  aws.String(msg),
			TopicArn: aws.String(cfg.AWS.TargetSNSTopicARN),
			Subject:  aws.String(subject),
		})
		if err != nil {
			log.Error("Failed to publish to target SNS topic: %s", err.Error())
			return events.SNSEvent{Records: []events.SNSEventRecord{}}, fmt.Errorf("publish error: %w", err)
		}
	}

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
