package deleter

import (
	"context"
	"time"

	"github.com/milagre/zote/go/zamqp"
	"github.com/milagre/zote/go/zlog"
	"github.com/milagre/zote/go/zorm"

	"github.com/milagre/zote/examples/project/src/service/internal/models"
)

// Worker consumes delete messages and removes rows from the database.
type Worker struct {
	repo zorm.Repository
}

// New returns a worker that hard-deletes items by id from messages.
func New(repo zorm.Repository) *Worker {
	return &Worker{repo: repo}
}

// Process implements the AMQP delivery handler.
func (w *Worker) Process(ctx context.Context, _ zamqp.Publisher, msg zamqp.Delivery) {
	logger := zlog.FromContext(ctx)
	payload := DeleteItemMessage{}
	if err := msg.Parse(&payload); err != nil {
		logger.Warnf("parse message: %v", err)
		_ = msg.Reject(ctx)
		return
	}
	if payload.ItemID == "" {
		logger.Warnf("empty item_id")
		_ = msg.Reject(ctx)
		return
	}

	err := zorm.Delete(ctx, w.repo, []*models.Item{{ID: payload.ItemID}}, zorm.DeleteOptions{})
	if err != nil {
		logger.Errorf("delete item: %v", err)
		retryOrFatal(ctx, msg, logger)
		return
	}
	logger.Infof("hard-deleted item %s", payload.ItemID)
	_ = msg.Ack(ctx)
}

func retryOrFatal(ctx context.Context, msg zamqp.Delivery, logger zlog.Logger) {
	var err error
	if msg.Attempt() <= 3 {
		err = msg.RetryDelayed(ctx, time.Duration(msg.Attempt())*time.Second)
	} else {
		err = msg.Fatal(ctx)
	}
	if err != nil {
		logger.Errorf("retry/fatal: %v", err)
	}
	_ = msg.Ack(ctx)
}
