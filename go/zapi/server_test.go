package zapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zstats"
	"github.com/milagre/zote/go/zstats/zprometheus"
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

func TestServeHTTP_PrometheusAfterNotFoundThenMatchedRoute(t *testing.T) {
	ctx := context.Background()
	prom := zprometheus.NewAdapter()
	stats := zstats.NewStats(prom)
	ctx = zstats.Context(ctx, stats)

	root := testRoute{
		name: "root",
		path: "",
		methods: Methods{
			http.MethodGet: {
				Handler: func(Request) ResponseBuilder {
					return BasicResponse(http.StatusOK, nil, []byte(`ok`))
				},
			},
		},
	}
	assets := testRoute{
		name: "asset_list",
		path: "/assets",
		methods: Methods{
			http.MethodPost: {
				Handler: func(Request) ResponseBuilder {
					return BasicResponse(http.StatusCreated, nil, []byte(`{}`))
				},
			},
		},
	}

	srv, err := NewServer(ctx, []Route{&root, &assets})
	require.NoError(t, err)

	reqMissing := httptest.NewRequest(http.MethodGet, "/missing", nil)
	reqMissing = reqMissing.WithContext(ctx)
	recMissing := httptest.NewRecorder()
	srv.handler.ServeHTTP(recMissing, reqMissing)

	reqAssets := httptest.NewRequest(http.MethodPost, "/assets", nil)
	reqAssets = reqAssets.WithContext(ctx)
	recAssets := httptest.NewRecorder()
	srv.handler.ServeHTTP(recAssets, reqAssets)
}

type testRoute struct {
	name    string
	path    string
	methods Methods
}

func (r *testRoute) Name() string     { return r.name }
func (r *testRoute) Path() string     { return r.path }
func (r *testRoute) Methods() Methods { return r.methods }

func TestContextWithTraceFromRequest(t *testing.T) {
	t.Parallel()

	base := context.Background()

	t.Run("sets correlation from header", func(t *testing.T) {
		t.Parallel()
		h := http.Header{}
		h.Set(ztrace.HeaderCorrelationID, "  correlation-abc  ")
		ctx := contextWithCorrelationID(base, h)
		id, ok := ztrace.ID(ctx)
		assert.True(t, ok)
		assert.Equal(t, "correlation-abc", id)
	})

	t.Run("no header generates correlation", func(t *testing.T) {
		t.Parallel()
		ctx := contextWithCorrelationID(base, http.Header{})
		id, ok := ztrace.ID(ctx)
		assert.True(t, ok)
		_, err := uuid.Parse(id)
		assert.NoError(t, err)
	})

	t.Run("blank header generates correlation", func(t *testing.T) {
		t.Parallel()
		h := http.Header{}
		h.Set(ztrace.HeaderCorrelationID, "   ")
		ctx := contextWithCorrelationID(base, h)
		id, ok := ztrace.ID(ctx)
		assert.True(t, ok)
		_, err := uuid.Parse(id)
		assert.NoError(t, err)
	})
}
