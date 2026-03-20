package zapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/milagre/zote/go/ztrace"
)

func TestIsParam(t *testing.T) {
	for name, data := range map[string]struct {
		input        string
		expected     bool
		expectedName string
	}{
		"string": {
			input:    "users",
			expected: false,
		},
		"variable": {
			input:        "{user_id}",
			expected:     true,
			expectedName: "user_id",
		},
	} {
		t.Run(name, func(t *testing.T) {
			name, ok := isParam(data.input)
			assert.Equal(t, data.expected, ok)
			assert.Equal(t, data.expectedName, name)
		})
	}
}

func TestContextWithTraceFromRequest(t *testing.T) {
	t.Parallel()

	base := context.Background()

	t.Run("sets trace from header", func(t *testing.T) {
		t.Parallel()
		h := http.Header{}
		h.Set(ztrace.HeaderTraceID, "  trace-abc  ")
		ctx := contextWithTraceID(base, h)
		id, ok := ztrace.ID(ctx)
		assert.True(t, ok)
		assert.Equal(t, "trace-abc", id)
	})

	t.Run("no header generates trace", func(t *testing.T) {
		t.Parallel()
		ctx := contextWithTraceID(base, http.Header{})
		id, ok := ztrace.ID(ctx)
		assert.True(t, ok)
		_, err := uuid.Parse(id)
		assert.NoError(t, err)
	})

	t.Run("blank header generates trace", func(t *testing.T) {
		t.Parallel()
		h := http.Header{}
		h.Set(ztrace.HeaderTraceID, "   ")
		ctx := contextWithTraceID(base, h)
		id, ok := ztrace.ID(ctx)
		assert.True(t, ok)
		_, err := uuid.Parse(id)
		assert.NoError(t, err)
	})
}
