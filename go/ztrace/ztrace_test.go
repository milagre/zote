package ztrace

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCorrelationIDContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, ok := ID(ctx)
	require.False(t, ok)

	id := NewID()
	require.NotEmpty(t, id)

	ctx = Context(ctx, id)
	got, ok := ID(ctx)
	require.True(t, ok)
	require.Equal(t, id, got)
}

func TestOutgoingCorrelationID(t *testing.T) {
	t.Parallel()

	t.Run("from context", func(t *testing.T) {
		t.Parallel()
		ctx := Context(context.Background(), "correlation-1")
		require.Equal(t, "correlation-1", OutgoingCorrelationID(ctx))
	})

	t.Run("generated when missing", func(t *testing.T) {
		t.Parallel()
		id := OutgoingCorrelationID(context.Background())
		_, err := uuid.Parse(id)
		require.NoError(t, err)
	})
}

func TestApplyHTTPRequestHeader(t *testing.T) {
	t.Parallel()

	ctx := Context(context.Background(), "correlation-xyz")
	req, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	require.NoError(t, err)

	ApplyHTTPRequestHeader(ctx, req)
	require.Equal(t, "correlation-xyz", req.Header.Get(HeaderCorrelationID))
}
