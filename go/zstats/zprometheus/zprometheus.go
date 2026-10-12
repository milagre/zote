package zprometheus

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"

	"github.com/milagre/zote/go/zlog"
	"github.com/milagre/zote/go/zstats"
)

var _ zstats.Adapter = &Adapter{}

// metricVecInfo holds the state for a metric type (counter, gauge, or histogram)
type metricVecInfo struct {
	vectors sync.Map // map[string]registeredVec
	mu      sync.Mutex
	prefix  string // "counter", "gauge", or "histogram"
	helpFmt string // Format string for help text
}

// registeredVec is a metric vector together with the label names it was
// registered with.
type registeredVec struct {
	collector  prometheus.Collector
	labelNames []string
}

type Adapter struct {
	registry *prometheus.Registry
	counter  *metricVecInfo
	gauge    *metricVecInfo
	hist     *metricVecInfo
}

// NewAdapter creates a new Prometheus adapter that implements zstats.Adapter.
func NewAdapter() *Adapter {
	return &Adapter{
		registry: prometheus.NewRegistry(),
		counter: &metricVecInfo{
			prefix:  "counter",
			helpFmt: "Counter metric: %s",
		},
		gauge: &metricVecInfo{
			prefix:  "gauge",
			helpFmt: "Gauge metric: %s",
		},
		hist: &metricVecInfo{
			prefix:  "histogram",
			helpFmt: "Histogram metric: %s",
		},
	}
}

// getOrCreateCounterVec gets or creates a CounterVec for the given metric name.
func (a *Adapter) getOrCreateCounterVec(name string, labelNames []string) (*prometheus.CounterVec, []string) {
	vec := a.getOrCreateVec(a.counter, name, labelNames, func(labelNames []string) prometheus.Collector {
		return prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: a.prometheusName(name),
				Help: fmt.Sprintf(a.counter.helpFmt, name),
			},
			labelNames,
		)
	})
	return vec.collector.(*prometheus.CounterVec), vec.labelNames
}

// getOrCreateGaugeVec gets or creates a GaugeVec for the given metric name.
func (a *Adapter) getOrCreateGaugeVec(name string, labelNames []string) (*prometheus.GaugeVec, []string) {
	vec := a.getOrCreateVec(a.gauge, name, labelNames, func(labelNames []string) prometheus.Collector {
		return prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: a.prometheusName(name),
				Help: fmt.Sprintf(a.gauge.helpFmt, name),
			},
			labelNames,
		)
	})
	return vec.collector.(*prometheus.GaugeVec), vec.labelNames
}

// getOrCreateHistogramVec gets or creates a HistogramVec for the given metric name.
func (a *Adapter) getOrCreateHistogramVec(name string, labelNames []string) (*prometheus.HistogramVec, []string) {
	vec := a.getOrCreateVec(a.hist, name, labelNames, func(labelNames []string) prometheus.Collector {
		return prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    a.prometheusName(name),
				Help:    fmt.Sprintf(a.hist.helpFmt, name),
				Buckets: prometheus.DefBuckets, // Default buckets: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
			},
			labelNames,
		)
	})
	return vec.collector.(*prometheus.HistogramVec), vec.labelNames
}

// getOrCreateVec is a generic helper that gets or creates a metric vector.
//
// Prometheus fixes a metric's label names when it is registered, so the label
// names of the first observation of a metric become its label names for the
// life of the adapter. Later observations are conformed to them by
// getLabelsForCanonicalSet rather than registering a conflicting vector.
func (a *Adapter) getOrCreateVec(info *metricVecInfo, name string, labelNames []string, createVec func([]string) prometheus.Collector) registeredVec {
	key := fmt.Sprintf("%s:%s", info.prefix, name)

	if vec, ok := info.vectors.Load(key); ok {
		return vec.(registeredVec)
	}

	info.mu.Lock()
	defer info.mu.Unlock()

	if vec, ok := info.vectors.Load(key); ok {
		return vec.(registeredVec)
	}

	vec := registeredVec{
		collector:  createVec(labelNames),
		labelNames: labelNames,
	}

	a.registry.MustRegister(vec.collector)
	info.vectors.Store(key, vec)
	return vec
}

func (a *Adapter) prometheusName(name string) string {
	return MetricName(name)
}

// MetricName converts a dot-delimited zstats name into the Prometheus metric
// name this adapter registers. Every character that is invalid in a Prometheus
// metric name is replaced with '_' (so dots and hyphens alike become '_'), and
// a leading character that cannot start a name is additionally prefixed with
// '_'. The result is always a valid legacy metric name, which is exactly what
// ends up stored after scraping — Prometheus underscore-escapes any invalid
// character (e.g. the hyphen in a workload name) on exposition, so emitting the
// already-escaped name here keeps registration, exposition, and storage in sync.
//
// It is exported so callers that must reconstruct a stored metric name out of
// band — e.g. a deployer generating a PromQL query for a metric this adapter
// will emit — share the exact naming rule instead of duplicating it.
func MetricName(name string) string {
	var b strings.Builder
	for i, r := range name {
		if i == 0 && !isNameStart(r) {
			b.WriteByte('_')
		}
		b.WriteRune(sanitizeNameChar(r))
	}

	return b.String()
}

// sanitizeNameChar maps a rune to itself when it is valid within a Prometheus
// metric name and to '_' otherwise.
func sanitizeNameChar(r rune) rune {
	if isNameContinuation(r) {
		return r
	}

	return '_'
}

func isNameStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == ':'
}

func isNameContinuation(r rune) bool {
	return isNameStart(r) || (r >= '0' && r <= '9')
}

// getLabelNamesAndValues extracts sorted label names and values from tags.
func (a *Adapter) getLabelNamesAndValues(tags zstats.Tags) ([]string, []string) {
	if len(tags) == 0 {
		return []string{}, []string{}
	}

	// Extract and sort label names
	names := make([]string, 0, len(tags))
	tagMap := make(map[string]string, len(tags))
	for k, v := range tags {
		names = append(names, k)
		tagMap[k] = v
	}

	// Sort label names
	sort.Strings(names)

	// Reorder values to match sorted names
	sortedValues := make([]string, len(names))
	for i, name := range names {
		sortedValues[i] = tagMap[name]
	}

	return names, sortedValues
}

// getLabelsForCanonicalSet creates a Labels map with exactly the canonical label
// names, using values from tags where available. Tags outside the canonical set
// are dropped. Missing or empty values become "unknown", since Prometheus
// requires every registered label to have a value.
func (a *Adapter) getLabelsForCanonicalSet(canonicalLabels []string, tags zstats.Tags) prometheus.Labels {
	labels := make(prometheus.Labels, len(canonicalLabels))
	for _, name := range canonicalLabels {
		if value, ok := tags[name]; ok && value != "" {
			labels[name] = value
		} else {
			// Prometheus doesn't allow empty label values, so we use "unknown"
			// as a standard sentinel value for missing labels
			labels[name] = "unknown"
		}
	}
	return labels
}

// Count implements zstats.Adapter.Count
func (a *Adapter) Count(name string, value float64, tags zstats.Tags) {
	labelNames, _ := a.getLabelNamesAndValues(tags)
	vec, canonicalLabels := a.getOrCreateCounterVec(name, labelNames)
	vec.With(a.getLabelsForCanonicalSet(canonicalLabels, tags)).Add(value)
}

// Gauge implements zstats.Adapter.Gauge
func (a *Adapter) Gauge(name string, value float64, tags zstats.Tags) {
	labelNames, _ := a.getLabelNamesAndValues(tags)
	vec, canonicalLabels := a.getOrCreateGaugeVec(name, labelNames)
	vec.With(a.getLabelsForCanonicalSet(canonicalLabels, tags)).Set(value)
}

// Timer implements zstats.Adapter.Timer
func (a *Adapter) Timer(name string, cb func(), tags zstats.Tags) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Nanoseconds()
		labelNames, _ := a.getLabelNamesAndValues(tags)
		vec, canonicalLabels := a.getOrCreateHistogramVec(name, labelNames)
		vec.With(a.getLabelsForCanonicalSet(canonicalLabels, tags)).Observe(float64(duration))
	}()
	cb()
}

// metricsHandler creates an HTTP handler that serves Prometheus metrics.
func (a *Adapter) metricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gatherer := prometheus.Gatherer(a.registry)
		mfs, err := gatherer.Gather()
		if err != nil {
			http.Error(w, fmt.Sprintf("error gathering metrics: %v", err), http.StatusInternalServerError)
			return
		}

		contentType := expfmt.Negotiate(r.Header)
		w.Header().Set("Content-Type", string(contentType))
		enc := expfmt.NewEncoder(w, contentType)

		for _, mf := range mfs {
			if err := enc.Encode(mf); err != nil {
				http.Error(w, fmt.Sprintf("error encoding metrics: %v", err), http.StatusInternalServerError)
				return
			}
		}
	})
}

// Start starts an HTTP server to expose Prometheus metrics.
// The server listens on the given address (e.g., ":8080") and serves metrics at /metrics.
func (a *Adapter) Start(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", a.metricsHandler())

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			zlog.FromContext(ctx).Errorf("Prometheus metrics server stopped unexpectedly: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		server.Shutdown(shutdownCtx)
	}()

	return nil
}
