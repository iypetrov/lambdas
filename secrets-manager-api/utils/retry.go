package utils

import (
	"time"
	"context"

	"github.com/cenkalti/backoff/v5"
)

func BackoffRetry[T any](ctx context.Context, op func() (T, error)) (T, error) {
	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.InitialInterval = 1 * time.Second
	backoffConfig.MaxInterval = 10 * time.Second
	backoffConfig.Reset()
	return backoff.Retry(ctx, op, backoff.WithBackOff(backoffConfig))
}
