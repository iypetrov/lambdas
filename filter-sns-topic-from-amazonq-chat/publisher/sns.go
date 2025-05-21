package publisher

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/iypetrov/lambdas/filter-sns-topic-from-amazonq-chat/config"
	"github.com/iypetrov/lambdas/filter-sns-topic-from-amazonq-chat/logger"
)

var (
	_ Publisher = (*SNSPublisher)(nil)
)

type SNSPublisher struct {
	logger            logger.Logger
	snsClient         *sns.Client
	TargetSNSTopicARN string
}

func NewSNSPublisher(
	ctx context.Context,
	log logger.Logger,
	cfg config.Config,
	targetSNSTopicARN string,
) *SNSPublisher {
	awsCfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Error("Failed to load AWS config: %s", err.Error())
		return nil
	}

	snsClient := sns.NewFromConfig(awsCfg)
	return &SNSPublisher{
		logger:            log,
		snsClient:         snsClient,
		TargetSNSTopicARN: targetSNSTopicARN,
	}
}

func (p *SNSPublisher) Publish(ctx context.Context, message string) error {
	_, err := p.snsClient.Publish(ctx, &sns.PublishInput{
		Message:  aws.String(message),
		TopicArn: aws.String(p.TargetSNSTopicARN),
	})
	return err
}
