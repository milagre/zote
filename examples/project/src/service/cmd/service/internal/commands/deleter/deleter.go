package deleter

import (
	"context"
	"fmt"

	"github.com/milagre/zote/go/zamqp/zamqpcmd"
	"github.com/milagre/zote/go/zcmd"
	"github.com/milagre/zote/go/zlog"
	"github.com/milagre/zote/go/zsig"

	"github.com/milagre/zote/examples/project/src/service/cmd/service/internal/app"
	"github.com/milagre/zote/examples/project/src/service/internal/workers/deleter"
)

type aspect struct {
	consumer zamqpcmd.DirectConsumerAspect
}

func (a aspect) Apply(c zcmd.Configurable) {
	a.consumer.Apply(c)
}

var Aspect = aspect{
	consumer: zamqpcmd.NewDirectConsumerAspect("core"),
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

	w := deleter.New(repo)

	consumer, err := Aspect.consumer.Consumer(
		env,
		deleter.Declarations,
		deleter.QueueName,
		w.Process,
	)
	if err != nil {
		return fmt.Errorf("consumer: %w", err)
	}

	sigCtx, cancel := zsig.Listen(ctx, zsig.Callbacks{})
	defer cancel()

	go func() {
		<-sigCtx.Done()
		logger.Info("shutting down deleter")
	}()

	return consumer.Start(sigCtx, ctx)
}
