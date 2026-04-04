package api

import (
	"context"
	"fmt"
	"time"

	"github.com/milagre/zote/go/zamqp/zamqpcmd"
	"github.com/milagre/zote/go/zapi"
	"github.com/milagre/zote/go/zcmd"
	"github.com/milagre/zote/go/zcmd/zaspect"
	"github.com/milagre/zote/go/zlog"
	"github.com/milagre/zote/go/zsig"

	"github.com/milagre/zote/examples/project/src/service/cmd/service/internal/app"
	"github.com/milagre/zote/examples/project/src/service/internal/api/routes"
)

type aspect struct {
	listen    zaspect.Listen
	publisher zamqpcmd.PublisherAspect
}

func (a aspect) Apply(c zcmd.Configurable) {
	a.listen.Apply(c)
	a.publisher.Apply(c)
}

var Aspect = aspect{
	listen:    zaspect.NewListenAspect(""),
	publisher: zamqpcmd.NewPublisherAspect("core"),
}

func Run(ctx context.Context, env zcmd.Env) error {
	ctx, err := app.Aspect.Context(ctx, env)
	if err != nil {
		return fmt.Errorf("context: %w", err)
	}
	logger := zlog.FromContext(ctx)

	repo, conn, err := app.Aspect.Repository(env)
	if err != nil {
		return fmt.Errorf("repository: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if err := app.EnsureSchema(ctx, conn); err != nil {
		return err
	}

	pub := Aspect.publisher.Publisher(env)

	srv, err := zapi.NewServer(ctx, routes.Routes(repo, pub),
		zapi.ServerOptionDefaultOptionsRequest(func(req zapi.Request) zapi.ResponseBuilder {
			return zapi.Response200OK()
		}),
	)
	if err != nil {
		return fmt.Errorf("zapi server: %w", err)
	}
	srv.AddMiddleware(zapi.NewCompressionMiddleware())

	sigCtx, cancel := zsig.Listen(ctx, zsig.Callbacks{})
	defer cancel()

	go func() {
		<-sigCtx.Done()
		logger.Info("shutting down api")
		c, cancelShutdown := context.WithTimeout(ctx, 60*time.Second)
		defer cancelShutdown()
		_ = srv.Shutdown(c)
	}()

	addr := Aspect.listen.ListenAddr(env)
	logger.Infof("listening on %s", addr)
	if err := srv.ListenAndServe(addr); err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}
