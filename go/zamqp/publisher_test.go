package zamqp

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/ztrace"
)

func TestMergePublishHeaders_JobIDFromIDs(t *testing.T) {
	t.Parallel()

	ctx := Context(context.Background(), "job-from-ctx", "msg-from-ctx")
	msg := NewRawMessage([]byte("x"), "text/plain", AnonymousExchange, MessageOptions{})
	h := mergePublishHeaders(ctx, msg)

	require.Equal(t, "job-from-ctx", h[headerJobID])
	require.NotEmpty(t, h[headerMessageID])
	_, err := uuid.Parse(h[headerMessageID].(string))
	require.NoError(t, err)

	jid, mid, ok := IDs(ctx)
	require.True(t, ok)
	require.Equal(t, "job-from-ctx", jid)
	require.Equal(t, "msg-from-ctx", mid)
}

func TestMergePublishHeaders_GeneratesJobIDAndTraceWhenMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	msg := NewRawMessage([]byte("x"), "text/plain", AnonymousExchange, MessageOptions{})
	h := mergePublishHeaders(ctx, msg)

	_, err := uuid.Parse(h[headerJobID].(string))
	require.NoError(t, err)
	_, err = uuid.Parse(h[headerCorrelationID].(string))
	require.NoError(t, err)

	_, _, ok := IDs(ctx)
	require.False(t, ok)
	_, ok = ztrace.ID(ctx)
	require.False(t, ok)
}

func TestMergePublishHeaders_MessageOptionsJobIDOverridesContext(t *testing.T) {
	t.Parallel()

	ctx := Context(context.Background(), "job-ctx", "msg-ctx")
	msg := NewRawMessage([]byte("x"), "text/plain", AnonymousExchange, MessageOptions{JobID: "job-opt"})
	h := mergePublishHeaders(ctx, msg)

	require.Equal(t, "job-opt", h[headerJobID])
	jid, mid, ok := IDs(ctx)
	require.True(t, ok)
	require.Equal(t, "job-ctx", jid)
	require.Equal(t, "msg-ctx", mid)
}

func TestMergePublishHeaders_MessageOptionsJobID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	msg := NewRawMessage([]byte("x"), "text/plain", AnonymousExchange, MessageOptions{JobID: "job-opt"})
	h := mergePublishHeaders(ctx, msg)

	require.Equal(t, "job-opt", h[headerJobID])
}

func TestMergePublishHeaders_HeadersJobIDIgnoredWhenJobIDOptionSet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	msg := NewRawMessage([]byte("x"), "text/plain", AnonymousExchange, MessageOptions{
		Headers: Headers{headerJobID: "job-header"},
		JobID:   "job-opt",
	})
	h := mergePublishHeaders(ctx, msg)

	require.Equal(t, "job-opt", h[headerJobID])
}

func TestMergePublishHeaders_HeadersJobIDIgnoredUsesContext(t *testing.T) {
	t.Parallel()

	ctx := Context(context.Background(), "job-from-ctx", "msg-from-ctx")
	msg := NewRawMessage([]byte("x"), "text/plain", AnonymousExchange, MessageOptions{
		Headers: Headers{headerJobID: "job-header"},
	})
	h := mergePublishHeaders(ctx, msg)

	require.Equal(t, "job-from-ctx", h[headerJobID])
}

func TestMergePublishHeaders_HeadersJobIDIgnoredGeneratesUUID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	msg := NewRawMessage([]byte("x"), "text/plain", AnonymousExchange, MessageOptions{
		Headers: Headers{headerJobID: "job-header"},
	})
	h := mergePublishHeaders(ctx, msg)

	_, err := uuid.Parse(h[headerJobID].(string))
	require.NoError(t, err)
	require.NotEqual(t, "job-header", h[headerJobID])
}

func TestMergePublishHeaders_TraceFromContext(t *testing.T) {
	t.Parallel()

	ctx := ztrace.Context(context.Background(), "correlation-1")
	msg := NewRawMessage([]byte("x"), "text/plain", AnonymousExchange, MessageOptions{})
	h := mergePublishHeaders(ctx, msg)

	require.Equal(t, "correlation-1", h[headerCorrelationID])
	cid, ok := ztrace.ID(ctx)
	require.True(t, ok)
	require.Equal(t, "correlation-1", cid)
}

func TestMergePublishHeaders_ExplicitCorrelationHeaderPreservedJobFromContext(t *testing.T) {
	t.Parallel()

	ctx := Context(context.Background(), "job-ctx", "msg-ctx")
	msg := NewRawMessage([]byte("x"), "text/plain", AnonymousExchange, MessageOptions{
		Headers: Headers{
			headerJobID:         "job-header",
			headerCorrelationID: "correlation-header",
		},
	})
	h := mergePublishHeaders(ctx, msg)

	require.Equal(t, "job-ctx", h[headerJobID])
	require.Equal(t, "correlation-header", h[headerCorrelationID])
	jid, mid, ok := IDs(ctx)
	require.True(t, ok)
	require.Equal(t, "job-ctx", jid)
	require.Equal(t, "msg-ctx", mid)
	_, ok = ztrace.ID(ctx)
	require.False(t, ok)
}
