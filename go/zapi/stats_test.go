package zapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zstats"
)

// recordingAdapter keeps the latest value seen for each gauge.
type recordingAdapter struct {
	mu     sync.Mutex
	gauges map[string]float64
}

func newRecordingAdapter() *recordingAdapter {
	return &recordingAdapter{gauges: map[string]float64{}}
}

func (a *recordingAdapter) Count(string, float64, zstats.Tags) {}

func (a *recordingAdapter) Gauge(name string, value float64, _ zstats.Tags) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.gauges[name] = value
}

func (a *recordingAdapter) Timer(_ string, cb func(), _ zstats.Tags) { cb() }

func (a *recordingAdapter) gauge(name string) (float64, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	v, ok := a.gauges[name]
	return v, ok
}

func TestConcurrencyStatName(t *testing.T) {
	assert.Equal(t, "app.apps.api.zapi.concurrency", ConcurrencyStatName("app.apps.api"))
}

// A request counts as in flight from the moment it is routed until its response
// has been written, so the gauge covers the whole time a replica is occupied and
// not just the handler.
func TestInFlightSpansTheWholeRequest(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})

	srv := blockingServer(t, entered, release)

	done := make(chan struct{})
	go func() {
		defer close(done)
		serve(srv)
	}()

	<-entered
	assert.Equal(t, int64(1), srv.inFlight.Load())

	close(release)
	<-done
	assert.Equal(t, int64(0), srv.inFlight.Load())
}

// The gauge is republished on a timer rather than at request boundaries: a
// scraped gauge holds its last value, so a server that went idle would keep
// reporting the concurrency it had when its final request finished.
func TestReportConcurrencyRepublishesWhileIdle(t *testing.T) {
	adapter := newRecordingAdapter()
	ctx := zstats.Context(context.Background(), zstats.NewStats(adapter))

	srv, err := NewServer(ctx, []Route{&testRoute{name: "root", path: ""}})
	require.NoError(t, err)

	reportCtx, stop := context.WithCancel(ctx)
	defer stop()
	go srv.reportConcurrency(reportCtx, time.Millisecond)

	name := ConcurrencyStatName("")

	srv.inFlight.Store(3)
	assertGauge(t, adapter, name, 3)

	srv.inFlight.Store(0)
	assertGauge(t, adapter, name, 0)
}

func blockingServer(t *testing.T, entered, release chan struct{}) *server {
	t.Helper()

	root := testRoute{
		name: "root",
		path: "",
		methods: Methods{
			http.MethodGet: {
				Handler: func(Request) ResponseBuilder {
					close(entered)
					<-release

					return BasicResponse(http.StatusOK, nil, []byte(`ok`))
				},
			},
		},
	}

	srv, err := NewServer(context.Background(), []Route{&root})
	require.NoError(t, err)

	return srv
}

func serve(srv *server) {
	srv.handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func assertGauge(t *testing.T, adapter *recordingAdapter, name string, want float64) {
	t.Helper()

	assert.Eventually(t, func() bool {
		got, ok := adapter.gauge(name)
		return ok && got == want
	}, time.Second, time.Millisecond)
}
