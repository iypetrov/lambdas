package config

import (
	"context"
	"log"
	"os"
)

type Environment string

const (
	Local Environment = "local"
	Prod  Environment = "prod"
)

func (e Environment) IsValid() bool {
	switch e {
	case Local, Prod:
		return true
	default:
		return false
	}
}

var (
	key = "CONFIG"
)

type Config struct {
	App struct {
		Env Environment
	}
	Slack struct {
		ChannelIDArn   string
		BotTokenArn	   string
	}
}

func Inject(ctx context.Context, cfg Config) context.Context {
	return context.WithValue(ctx, key, cfg)
}

func Get(ctx context.Context) Config {
	c, ok := ctx.Value(key).(Config)
	if !ok {
		log.Fatal("couldn't get config from context")
	}
	return c
}

func New() Config {
	cfg := Config{}

	cfg.App.Env = Environment(os.Getenv("APP_ENV"))
	if !cfg.App.Env.IsValid() {
		cfg.App.Env = Prod
	}

	cfg.Slack.ChannelIDArn = os.Getenv("SLACK_CHANNEL_ID_ARN")
	cfg.Slack.BotTokenArn = os.Getenv("SLACK_BOT_TOKEN_ARN")

	return cfg
}
