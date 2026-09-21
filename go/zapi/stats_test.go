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

// recordingAdapter sums every value counted under each name.
type recordingAdapter struct {
	mu     sync.Mutex
	counts map[string]float64
}

func newRecordingAdapter() *recordingAdapter {
	return &recordingAdapter{counts: map[string]float64{}}
}

func (a *recordingAdapter) Count(name string, value float64, _ zstats.Tags) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.counts[name] += value
}

func (a *recordingAdapter) Gauge(string, float64, zstats.Tags) {}

func (a *recordingAdapter) Timer(_ string, cb func(), _ zstats.Tags) { cb() }

func (a *recordingAdapter) count(name string) float64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.counts[name]
}

func TestBusySecondsStatName(t *testing.T) {
	assert.Equal(t, "app.apps.my-api.zapi.busy_seconds", BusySecondsStatName("app.apps.my-api"))
}

// Busy time is the integral of in-flight requests over time, so its rate is the
// mean concurrency however short the requests are relative to the scrape.
func TestBusyClockIntegratesOverlappingRequests(t *testing.T) {
	var c busyClock
	t0 := time.Unix(0, 0)

	c.add(t0, 1)
	c.add(t0.Add(time.Second), 1)
	c.add(t0.Add(2*time.Second), -1)
	c.add(t0.Add(3*time.Second), -1)

	assert.Equal(t, 4*time.Second, c.total(t0.Add(10*time.Second)))
}

// A request still open counts toward busy time as it runs, not only once it
// completes, so a long request does not land as a single spike at its end.
func TestBusyClockCountsOpenRequests(t *testing.T) {
	var c busyClock
	t0 := time.Unix(0, 0)

	c.add(t0, 2)

	assert.Equal(t, 2*time.Second, c.total(t0.Add(time.Second)))
	assert.Equal(t, 6*time.Second, c.total(t0.Add(3*time.Second)))
}

// A request counts as in flight from the moment it is routed until its response
// has been written, so busy time covers the whole time a replica is occupied and
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
	assert.Equal(t, int64(1), srv.busy.current())

	close(release)
	<-done
	assert.Equal(t, int64(0), srv.busy.current())
}

// Every interval of busy time is published exactly once, including the tail
// accrued after the last tick, so the counter's total matches the clock's.
func TestReportBusyPublishesAllBusyTime(t *testing.T) {
	adapter := newRecordingAdapter()
	ctx := zstats.Context(context.Background(), zstats.NewStats(adapter))

	srv, err := NewServer(ctx, []Route{&testRoute{name: "root", path: ""}})
	require.NoError(t, err)

	reportCtx, stop := context.WithCancel(ctx)
	reported := make(chan struct{})
	go func() {
		defer close(reported)
		srv.reportBusy(reportCtx, time.Millisecond)
	}()

	name := BusySecondsStatName("")

	srv.busy.add(time.Now(), 1)
	assert.Eventually(t, func() bool { return adapter.count(name) > 0 }, time.Second, time.Millisecond)
	srv.busy.add(time.Now(), -1)

	stop()
	<-reported

	assert.InDelta(t, srv.busy.total(time.Now()).Seconds(), adapter.count(name), 1e-9)
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
