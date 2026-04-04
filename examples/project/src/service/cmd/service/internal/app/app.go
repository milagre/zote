package app

import (
	"context"
	"fmt"

	"github.com/milagre/zote/go/zcmd"
	"github.com/milagre/zote/go/zlog"
	"github.com/milagre/zote/go/zlog/zlogcmd"
	"github.com/milagre/zote/go/zlog/zlogrus"
	"github.com/milagre/zote/go/zorm/zormsql"
	"github.com/milagre/zote/go/zsql"
	"github.com/milagre/zote/go/zsql/zmysql"
	"github.com/milagre/zote/go/zstats"
	"github.com/milagre/zote/go/zstats/zstatscmd"
	"github.com/milagre/zote/go/zstats/zstatsd"

	"github.com/milagre/zote/examples/project/src/service/internal/models"
)

var Aspect = aspect{
	mysql: zmysql.NewAspect("core"),
	log:   zlogcmd.Aspect{},
	stats: zstatscmd.Aspect{},
	statsd: zstatsd.Aspect{},
}

type aspect struct {
	mysql  zmysql.Aspect
	log    zlogcmd.Aspect
	stats  zstatscmd.Aspect
	statsd zstatsd.Aspect
}

func (a aspect) Apply(c zcmd.Configurable) {
	a.log.Apply(c)
	a.stats.Apply(c)
	a.statsd.Apply(c)
	a.mysql.Apply(c)
}

func (a aspect) Context(ctx context.Context, env zcmd.Env) (context.Context, error) {
	logger := zlog.New(a.log.LogLevel(env), zlogrus.New(a.log.LogLevel(env), a.log.LogFormat(env)))
	ctx = zlog.Context(ctx, logger)

	statsdClient, err := a.statsd.Client(env)
	if err != nil {
		return ctx, fmt.Errorf("statsd client: %w", err)
	}
	statsdAdapter := zstatsd.NewAdapter(statsdClient)
	stats, err := a.stats.Configure(env, zstats.NewStats(statsdAdapter))
	if err != nil {
		return ctx, fmt.Errorf("stats: %w", err)
	}
	ctx = zstats.Context(ctx, stats)

	return ctx, nil
}

func (a aspect) Repository(env zcmd.Env) (*zormsql.Repository, zsql.Connection, error) {
	db, err := a.mysql.Connection(env, zmysql.DefaultOptions())
	if err != nil {
		return nil, nil, fmt.Errorf("mysql: %w", err)
	}
	repo := zormsql.NewRepository("core", db)
	models.AddMappings(repo)
	return repo, db, nil
}

func EnsureSchema(ctx context.Context, conn zsql.Connection) error {
	_, err := conn.Exec(ctx, `
CREATE TABLE IF NOT EXISTS items (
	id VARCHAR(36) NOT NULL PRIMARY KEY,
	created DATETIME(6) NOT NULL,
	modified DATETIME(6) NULL,
	name VARCHAR(255) NOT NULL,
	deleted TINYINT(1) NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
`)
	if err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}
	return nil
}
