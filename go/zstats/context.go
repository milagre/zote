package zstats

import (
	"context"
	"runtime/debug"

	"github.com/milagre/zote/go/zlog"
)

type contextKeyType string

const contextKey contextKeyType = "stats"

func FromContext(ctx context.Context) Stats {
	v := ctx.Value(contextKey)
	if v != nil {
		if s, ok := v.(Stats); ok {
			return s
		}
	}

	stackBytes := debug.Stack()
	stackString := string(stackBytes)
	partialLength := 1000
	if len(stackString) < partialLength {
		partialLength = len(stackString) - 1
	}
	stackStringPartial := stackString[0:partialLength]

	zlog.FromContext(ctx).Warnf("stats requested from context, but no stats are have been set: %s", stackStringPartial)

	return NewStats(NewNullAdapter())
}

func Context(ctx context.Context, stats Stats) context.Context {
	return context.WithValue(ctx, contextKey, stats)
}
