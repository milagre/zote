package zormtest

import (
	"context"

	"github.com/milagre/zote/go/zlog"
	"github.com/milagre/zote/go/zlog/zlogrus"
)

func makeContext(ctx context.Context) context.Context {
	ctx = zlog.Context(ctx, zlog.New(zlog.LevelDebug, zlogrus.New(zlog.LevelDebug)))
	return ctx
}
