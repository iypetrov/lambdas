package slack

import (
	"context"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/iypetrov/lambdas/queue-event-to-slack/config"
	"github.com/iypetrov/lambdas/queue-event-to-slack/logger"
	"github.com/slack-go/slack"
)

type Slack struct {
	api *slack.Client
	channelID string
	log logger.Logger
}

func NewSlack(ctx context.Context,cfg config.Config, log logger.Logger) (*Slack, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Error("Failed to load AWS config: %v", err)
		return nil, err
	}
	smc := secretsmanager.NewFromConfig(awsCfg)

	var channelID string
	var botToken string
	channelIDResp, err := smc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &cfg.Slack.ChannelIDArn,
	})
	if err != nil {
		return nil, err
	}
	channelID = *channelIDResp.SecretString

	botTokenResp, err := smc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &cfg.Slack.BotTokenArn,
	})
	if err != nil {
		return nil
	}
	botToken = *botTokenResp.SecretString

	return &Slack{
		api: slack.New(botToken),
		channelID: channelID,
		log: log,
	}, nil
}

func (s *Slack) SendMessage(msg string) error {
	_, _, err := s.api.PostMessage(
		s.channelID,
		slack.MsgOptionText(msg, false),
	)
	if err != nil {
		s.log.Error("failed to send message %v to Slack channel: %s", err, s.channelID)
		return err
	}
	s.log.Info("Message sent successfully to Slack channel: %s", s.channelID)

	return nil
}
