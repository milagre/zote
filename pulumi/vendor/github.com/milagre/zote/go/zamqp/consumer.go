package zamqp

import (
	"context"
)

type Consumer interface {
	// Does not return until shutdown. On context cancellation, waits for in-flight messages to finish. Message processing uses
	// context.WithoutCancel on ctx so work is not interrupted when shutdown begins. An error is only returned if the system fails to start
	// successfully.
	Start(ctx context.Context) error
}

type ConsumeFunc func(ctx context.Context, publisher Publisher, msg Delivery)
