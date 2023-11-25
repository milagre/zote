package zlog

import (
	"context"
	"fmt"
	"runtime/debug"
)

type contextKeyType string

const contextKey contextKeyType = "log"

func FromContext(ctx context.Context) Logger {
	v := ctx.Value(contextKey)
	if v != nil {
		if l, ok := v.(Logger); ok {
			return l
		}
	}

	fmt.Println("warn: logger requested but no logger is set")
	debug.PrintStack()

	return &logger{}
}

func Context(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, contextKey, logger)
}
