package logger

import (
	"context"
	"go.uber.org/zap"
	"log"
)

type KeyType string

var logKey KeyType = "LOGGER"

func Inject(ctx context.Context, log *zap.Logger) context.Context {
	return context.WithValue(ctx, "LOGGER", log)
}

func GetLoggerFromContext(ctx context.Context) *zap.Logger {
	c, ok := ctx.Value(logKey).(*zap.Logger)
	if !ok {
		log.Fatal("couldn't get logger from context")
	}
	return c
}
