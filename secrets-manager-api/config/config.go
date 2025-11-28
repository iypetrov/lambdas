package config

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
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

type contextKey string

var (
	key contextKey = "CONFIG"
)

type Config struct {
	App struct {
		Env Environment
	}
	S3 struct {
		Bucket string
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
	_ = godotenv.Load()

	cfg.App.Env = Environment(os.Getenv("APP_ENV"))
	if !cfg.App.Env.IsValid() {
		cfg.App.Env = Prod
	}

	cfg.S3.Bucket = os.Getenv("S3_BUCKET")

	return cfg
}
