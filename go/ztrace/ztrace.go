// Package ztrace carries distributed trace identifiers in context.Context.
// It is used by zamqp, zapi, and other packages to correlate work across requests and messages.
package ztrace

import (
	"context"

	"github.com/google/uuid"
	"github.com/milagre/zote/go/zlog"
)

// HeaderTraceID is the canonical application header for passing
// a trace ID over the wire.
const HeaderTraceID = "x-zote-trace-id"

type traceIDKeyType string

var traceIDKey traceIDKeyType = "trace_id"

// Context returns a derived context that carries traceID for ID.
func Context(ctx context.Context, traceID string) context.Context {
	zlog.FromContext(ctx).AddField("trace_id", traceID)
	return context.WithValue(ctx, traceIDKey, traceID)
}

// ID returns the trace ID from ctx, if set and non-empty.
func ID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(traceIDKey).(string)
	return v, ok && v != ""
}

// NewTraceID returns a new random trace identifier (UUID string).
func NewTraceID() string {
	return uuid.NewString()
}
