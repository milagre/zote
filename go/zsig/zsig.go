package zsig

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/milagre/zote/go/zlog"
)

type SignalFunc func()

type Callbacks struct {
	Reload SignalFunc
}

func Listen(ctx context.Context, cbs Callbacks) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	ch := make(chan os.Signal, 1)

	callbacks := map[os.Signal]func(){
		syscall.SIGTERM: cancel,
		syscall.SIGINT:  cancel,
	}

	signals := []os.Signal{
		syscall.SIGTERM,
		syscall.SIGINT,
	}

	if cbs.Reload != nil {
		signals = append(signals, syscall.SIGUSR1)
		callbacks[syscall.SIGUSR1] = cbs.Reload
	}

	signal.Notify(ch, signals...)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case sig := <-ch:
				if f, ok := callbacks[sig]; ok {
					zlog.FromContext(ctx).Infof("Signal %s received", sig)
					f()
				} else {
					zlog.FromContext(ctx).Infof("Signal %s ignored", sig)
				}

				if sig == syscall.SIGTERM || sig == syscall.SIGINT {
					signal.Ignore()
					return
				}
			}
		}
	}()

	return ctx, cancel
}
