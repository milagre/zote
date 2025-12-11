package zredis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type nullLogger struct{}

func (nullLogger) Printf(ctx context.Context, format string, v ...interface{}) {}

func init() {
	redis.SetLogger(nullLogger{})
}
