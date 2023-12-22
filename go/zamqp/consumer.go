package zamqp

import (
	"context"
)

type Consumer interface {
	// Does not return. On consumer context cancellation, will gracefully wait
	// for all messages to complete processing. The worker context should NOT be
	// canceled. An error is only returned if the system fails to start
	// successfully. Once the system has started, no error is possible.
	Start(processCtx context.Context, workerCtx context.Context) error
}

type ConsumeFunc func(ctx context.Context, msg Delivery)
