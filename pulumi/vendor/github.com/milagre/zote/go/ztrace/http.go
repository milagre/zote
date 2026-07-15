package ztrace

import (
	"context"
	"net/http"
)

// OutgoingCorrelationID returns the trace ID from ctx when set; otherwise a new NewCorrelationID().
// Use this (or ApplyHTTPRequestHeader) when propagating correlation on outbound calls.
func OutgoingCorrelationID(ctx context.Context) string {
	if id, ok := ID(ctx); ok {
		return id
	}
	return NewID()
}

// ApplyHTTPRequestHeader sets HeaderCorrelationID on req using OutgoingCorrelationID(ctx).
func ApplyHTTPRequestHeader(ctx context.Context, req *http.Request) {
	req.Header.Set(HeaderCorrelationID, OutgoingCorrelationID(ctx))
}
