package ztrace

import (
	"context"
	"net/http"
)

// OutgoingTraceID returns the trace ID from ctx when set; otherwise a new NewTraceID().
// Use this (or ApplyHTTPRequestHeader) when propagating correlation on outbound calls.
func OutgoingTraceID(ctx context.Context) string {
	if id, ok := ID(ctx); ok {
		return id
	}
	return NewTraceID()
}

// ApplyHTTPRequestHeader sets HeaderTraceID on req using OutgoingTraceID(ctx).
func ApplyHTTPRequestHeader(ctx context.Context, req *http.Request) {
	req.Header.Set(HeaderTraceID, OutgoingTraceID(ctx))
}
