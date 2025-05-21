package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/iypetrov/lambdas/filter-sns-topic-from-amazonq-chat/config"
	"github.com/iypetrov/lambdas/filter-sns-topic-from-amazonq-chat/logger"
	"github.com/iypetrov/lambdas/filter-sns-topic-from-amazonq-chat/publisher"
	"golang.org/x/sync/errgroup"
)

type AlarmMessage struct {
	AlarmName                          string       `json:"AlarmName"`
	AlarmDescription                   string       `json:"AlarmDescription"`
	AWSAccountID                       string       `json:"AWSAccountId"`
	AlarmConfigurationUpdatedTimestamp string       `json:"AlarmConfigurationUpdatedTimestamp"`
	NewStateValue                      string       `json:"NewStateValue"`
	NewStateReason                     string       `json:"NewStateReason"`
	StateChangeTime                    string       `json:"StateChangeTime"`
	Region                             string       `json:"Region"`
	AlarmArn                           string       `json:"AlarmArn"`
	OldStateValue                      string       `json:"OldStateValue"`
	OKActions                          []string     `json:"OKActions"`
	AlarmActions                       []string     `json:"AlarmActions"`
	InsufficientDataActions            []string     `json:"InsufficientDataActions"`
	Trigger                            AlarmTrigger `json:"Trigger"`
}

type AlarmTrigger struct {
	MetricName                       string      `json:"MetricName"`
	Namespace                        string      `json:"Namespace"`
	StatisticType                    string      `json:"StatisticType"`
	Statistic                        string      `json:"Statistic"`
	Unit                             *string     `json:"Unit"`
	Dimensions                       []Dimension `json:"Dimensions"`
	Period                           int         `json:"Period"`
	EvaluationPeriods                int         `json:"EvaluationPeriods"`
	ComparisonOperator               string      `json:"ComparisonOperator"`
	Threshold                        float64     `json:"Threshold"`
	TreatMissingData                 string      `json:"TreatMissingData"`
	EvaluateLowSampleCountPercentile string      `json:"EvaluateLowSampleCountPercentile"`
}

type Dimension struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func Handler(ctx context.Context, event events.SNSEvent) error {
	log := logger.Get(ctx)
	cfg := config.Get(ctx)
	snsPublisher := publisher.NewSNSPublisher(
		ctx,
		log,
		cfg,
		cfg.AWS.TargetSNSTopicARN,
	)

	log.Info("%d records were send to the %s", len(event.Records), cfg.AWS.TargetSNSTopicARN)
	var g errgroup.Group
	for _, record := range event.Records {
		var alarmMsg AlarmMessage
		err := json.Unmarshal([]byte(record.SNS.Message), &alarmMsg)
		if err != nil {
			log.Error("Failed to parse alarm message: %v", err)
			return err
		}
		g.Go(func() error {
			return snsPublisher.Publish(ctx, record.SNS.Message)
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
