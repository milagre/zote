package ztrace

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTraceIDContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, ok := ID(ctx)
	require.False(t, ok)

	id := NewTraceID()
	require.NotEmpty(t, id)

	ctx = Context(ctx, id)
	got, ok := ID(ctx)
	require.True(t, ok)
	require.Equal(t, id, got)
}

func TestOutgoingTraceID(t *testing.T) {
	t.Parallel()

	t.Run("from context", func(t *testing.T) {
		t.Parallel()
		ctx := Context(context.Background(), "trace-1")
		require.Equal(t, "trace-1", OutgoingTraceID(ctx))
	})

	t.Run("generated when missing", func(t *testing.T) {
		t.Parallel()
		id := OutgoingTraceID(context.Background())
		_, err := uuid.Parse(id)
		require.NoError(t, err)
	})
}

func TestApplyHTTPRequestHeader(t *testing.T) {
	t.Parallel()

	ctx := Context(context.Background(), "trace-xyz")
	req, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	require.NoError(t, err)

	ApplyHTTPRequestHeader(ctx, req)
	require.Equal(t, "trace-xyz", req.Header.Get(HeaderTraceID))
}
