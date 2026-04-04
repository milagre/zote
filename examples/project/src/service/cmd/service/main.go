package main

import (
	"context"

	"github.com/milagre/zote/go/zcmd"
	"github.com/milagre/zote/go/zlog"
	"github.com/milagre/zote/go/zlog/zlogrus"

	"github.com/milagre/zote/examples/project/src/service/cmd/service/internal/app"
	"github.com/milagre/zote/examples/project/src/service/cmd/service/internal/commands/api"
	"github.com/milagre/zote/examples/project/src/service/cmd/service/internal/commands/deleter"
)

func main() {
	ctx := context.Background()
	logger := zlog.New(zlog.LevelInfo, zlogrus.New(zlog.LevelTrace, zlog.FormatText))
	ctx = zlog.Context(ctx, logger)
	logger.Info("service starting")

	zcmd.NewApp(
		"service",
		"DEMO_",
		app.Aspect,
		map[string]zcmd.Command{
			"api": {
				Config: api.Aspect,
				Run:    api.Run,
			},
			"deleter": {
				Config: deleter.Aspect,
				Run:    deleter.Run,
			},
		},
	).Run(ctx)
}
