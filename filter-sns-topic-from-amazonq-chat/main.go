package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/iypetrov/lambdas/filter-sns-topic-from-amazonq-chat/config"
	"github.com/iypetrov/lambdas/filter-sns-topic-from-amazonq-chat/logger"
	"github.com/iypetrov/lambdas/filter-sns-topic-from-amazonq-chat/message"
	"github.com/iypetrov/lambdas/filter-sns-topic-from-amazonq-chat/publisher"
	"github.com/iypetrov/lambdas/filter-sns-topic-from-amazonq-chat/sanitizer"
	"golang.org/x/sync/errgroup"
)

func Handler(ctx context.Context, event events.SNSEvent) error {
	log := logger.Get(ctx)
	cfg := config.Get(ctx)
	snsPublisher := publisher.NewSNSPublisher(
		ctx,
		log,
		cfg,
		cfg.AWS.TargetSNSTopicARN,
	)

	sanitizer := sanitizer.NewChainedSanitizer(sanitizer.NewStandardSanitizers()...)

	log.Info("%d records were send to the %s", len(event.Records), cfg.AWS.TargetSNSTopicARN)
	var g errgroup.Group
	for _, record := range event.Records {
		var alarmMsg message.AlarmMessage
		err := json.Unmarshal([]byte(record.SNS.Message), &alarmMsg)
		if err != nil {
			log.Error("Failed to parse alarm message: %v", err)
		}

		log.Info("Alarm %s in state %s", alarmMsg.AlarmName, alarmMsg.NewStateValue)
		santizedAlarmMsg := sanitizer(alarmMsg)
		if santizedAlarmMsg == nil {
			log.Info("Alarm %s in state %s was dropped", alarmMsg.AlarmName, alarmMsg.NewStateValue)
			continue
		}

		pretty, err := json.MarshalIndent(alarmMsg, "", "  ")
		if err != nil {
			log.Error("Failed to marshal message back: %v", err)
			return err
		}

		g.Go(func() error {
			return snsPublisher.Publish(ctx, string(pretty))
		})
	}

	return g.Wait()
}

func main() {
	ctx := context.Background()
	cfg := config.New()
	log := logger.New(cfg)
	ctx = log.Inject(ctx)
	ctx = config.Inject(ctx, *cfg)

	lambda.StartWithOptions(Handler, lambda.WithContext(ctx))
}
