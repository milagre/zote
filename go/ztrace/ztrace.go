// Package ztrace carries distributed trace identifiers in context.Context.
// It is used by zamqp, zapi, and other packages to correlate work across requests and messages.
package ztrace

import (
	"context"

	"github.com/google/uuid"

	"github.com/milagre/zote/go/zlog"
)

// HeaderCorrelationID is the canonical application header for passing
// a correlation ID over the wire.
const HeaderCorrelationID = "x-zote-correlation-id"

type correlationIDKeyType string

var correlationIDKey correlationIDKeyType = "correlation_id"

// Context returns a derived context that carries correlationID for ID and attaches correlation_id to the
// zlog logger in ctx (via WithField so the value is visible to zlog.FromContext).
func Context(ctx context.Context, correlationID string) context.Context {
	ctx = zlog.Context(ctx, zlog.FromContext(ctx).WithField("correlation_id", correlationID))
	return context.WithValue(ctx, correlationIDKey, correlationID)
}

// ID returns the correlation ID from ctx, if set and non-empty.
func ID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(correlationIDKey).(string)
	return v, ok && v != ""
}

// NewID returns a new random correlation identifier (UUID string).
func NewID() string {
	return uuid.NewString()
}
