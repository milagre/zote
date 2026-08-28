package zamqp

import (
	"context"
	"maps"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zlog"
	"github.com/milagre/zote/go/zstats"
)

// The Stats in the context is the process-wide object, shared with every
// handler this consumer runs. Scoping the consumer's own metrics by mutating it
// renames every metric the process emits from then on.
func TestConsumerScopingLeavesTheProcessStatsAlone(t *testing.T) {
	t.Parallel()

	recorder := &recordingAdapter{}
	ctx := zstats.Context(testContext(), zstats.NewStats(recorder))

	consumerStats(ctx, "account-analyzer").Count("utilization", 1)
	zstats.FromContext(ctx).Count("llm.requests", 1)

	assert.Equal(t, []string{"zamqp.consumer.utilization", "llm.requests"}, recorder.names())
	assert.NotContains(t, recorder.tagsFor("llm.requests"), "queue")
}

func TestConsumerCountsItsOwnWorkBeneathTheConsumerPrefix(t *testing.T) {
	t.Parallel()

	recorder := &recordingAdapter{}
	ctx := zstats.Context(testContext(), zstats.NewStats(recorder))
	consumer := testConsumer(func(context.Context, Publisher, Delivery) {})

	consumer.consume(ctx, consumerStats(ctx, consumer.queueName), nil, oneDelivery(t))

	assert.Equal(t, []string{"zamqp.consumer.received", "zamqp.consumer.completed"}, recorder.names())
	assert.Equal(t, "account-analyzer", recorder.tagsFor("zamqp.consumer.received")["queue"])
}

func TestHandlerMetricsAreNotScopedToTheConsumer(t *testing.T) {
	t.Parallel()

	recorder := &recordingAdapter{}
	ctx := zstats.Context(testContext(), zstats.NewStats(recorder))
	consumer := testConsumer(func(ctx context.Context, _ Publisher, _ Delivery) {
		zstats.FromContext(ctx).Count("llm.requests", 1)
	})

	consumer.consume(ctx, consumerStats(ctx, consumer.queueName), nil, oneDelivery(t))

	assert.Contains(t, recorder.names(), "llm.requests")
	assert.NotContains(t, recorder.tagsFor("llm.requests"), "queue")
}

func testContext() context.Context {
	return zlog.Context(context.Background(), zlog.New(zlog.LevelError))
}

func testConsumer(process ConsumeFunc) *directConsumer {
	return &directConsumer{
		queueName:   "account-analyzer",
		concurrency: 1,
		process:     process,
		busyCounter: &atomic.Int64{},
	}
}

// oneDelivery is a closed channel holding a single message, so consume runs one
// message through and then reads the nil that ends its loop.
func oneDelivery(t *testing.T) chan Delivery {
	t.Helper()

	published, err := messageToPublishing(NewRawMessage(
		[]byte(`{"account_id":"137"}`),
		"application/json",
		Exchange{Name: "dest"},
		MessageOptions{},
	))
	require.NoError(t, err)

	deliveries := make(chan Delivery, 1)
	deliveries <- deliveryOf(published)
	close(deliveries)

	return deliveries
}

type recordedMetric struct {
	name  string
	value float64
	tags  zstats.Tags
}

type recordingAdapter struct {
	mu      sync.Mutex
	metrics []recordedMetric
}

func (a *recordingAdapter) Count(name string, value float64, tags zstats.Tags) {
	a.record(name, value, tags)
}

func (a *recordingAdapter) Gauge(name string, value float64, tags zstats.Tags) {
	a.record(name, value, tags)
}

func (a *recordingAdapter) Timer(_ string, cb func(), _ zstats.Tags) {
	cb()
}

func (a *recordingAdapter) record(name string, value float64, tags zstats.Tags) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.metrics = append(a.metrics, recordedMetric{name: name, value: value, tags: maps.Clone(tags)})
}

func (a *recordingAdapter) names() []string {
	a.mu.Lock()
	defer a.mu.Unlock()

	names := make([]string, 0, len(a.metrics))
	for _, metric := range a.metrics {
		names = append(names, metric.name)
	}

	return names
}

func (a *recordingAdapter) tagsFor(name string) zstats.Tags {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, metric := range a.metrics {
		if metric.name == name {
			return metric.tags
		}
	}

	return nil
}
