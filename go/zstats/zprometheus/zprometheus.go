package zprometheus

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
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
	vectors sync.Map // map[string]prometheus.Collector
	labels  sync.Map // map[string][]string - tracks label names per metric
	mu      sync.Mutex
	prefix  string // "counter", "gauge", or "histogram"
	helpFmt string // Format string for help text
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
func (a *Adapter) getOrCreateCounterVec(name string, labelNames []string) *prometheus.CounterVec {
	vec := a.getOrCreateVec(a.counter, name, labelNames, func(canonicalLabels []string) prometheus.Collector {
		return prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: a.prometheusName(name),
				Help: fmt.Sprintf(a.counter.helpFmt, name),
			},
			canonicalLabels,
		)
	})
	return vec.(*prometheus.CounterVec)
}

// getOrCreateGaugeVec gets or creates a GaugeVec for the given metric name.
func (a *Adapter) getOrCreateGaugeVec(name string, labelNames []string) *prometheus.GaugeVec {
	vec := a.getOrCreateVec(a.gauge, name, labelNames, func(canonicalLabels []string) prometheus.Collector {
		return prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: a.prometheusName(name),
				Help: fmt.Sprintf(a.gauge.helpFmt, name),
			},
			canonicalLabels,
		)
	})
	return vec.(*prometheus.GaugeVec)
}

// getOrCreateHistogramVec gets or creates a HistogramVec for the given metric name.
func (a *Adapter) getOrCreateHistogramVec(name string, labelNames []string) *prometheus.HistogramVec {
	vec := a.getOrCreateVec(a.hist, name, labelNames, func(canonicalLabels []string) prometheus.Collector {
		return prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    a.prometheusName(name),
				Help:    fmt.Sprintf(a.hist.helpFmt, name),
				Buckets: prometheus.DefBuckets, // Default buckets: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
			},
			canonicalLabels,
		)
	})
	return vec.(*prometheus.HistogramVec)
}

// getOrCreateVec is a generic helper that gets or creates a metric vector.
// It tracks all unique label names seen for this metric and uses their union.
func (a *Adapter) getOrCreateVec(info *metricVecInfo, name string, labelNames []string, createVec func([]string) prometheus.Collector) prometheus.Collector {
	key := fmt.Sprintf("%s:%s", info.prefix, name)

	// Try to get existing vector first (fast path)
	if vec, ok := info.vectors.Load(key); ok {
		// Update label names in case we see new ones, but don't block on it
		go func() {
			info.mu.Lock()
			defer info.mu.Unlock()
			a.getOrUpdateLabelNames(&info.labels, key, labelNames)
		}()
		return vec.(prometheus.Collector)
	}

	// Need to create vector - acquire lock
	info.mu.Lock()
	defer info.mu.Unlock()

	// Double-check after acquiring lock
	if vec, ok := info.vectors.Load(key); ok {
		a.getOrUpdateLabelNames(&info.labels, key, labelNames)
		return vec.(prometheus.Collector)
	}

	// Get or compute the canonical label names (union of all seen label names)
	canonicalLabels := a.getOrUpdateLabelNames(&info.labels, key, labelNames)

	// Create new vector with canonical label names
	vec := createVec(canonicalLabels)

	// Store and register
	info.vectors.Store(key, vec)
	a.registry.MustRegister(vec)
	return vec
}

// prometheusName converts a dot-delimited zstats name to a Prometheus-compatible name.
// Prometheus metric names should match [a-zA-Z_:][a-zA-Z0-9_:]* and dots are valid.
// We'll keep dots but ensure the name starts with a letter or underscore.
func (a *Adapter) prometheusName(name string) string {
	// Replace dots with underscores for Prometheus convention
	// But first ensure it starts with a letter or underscore
	result := ""
	for i, r := range name {
		if i == 0 {
			// First character must be letter or underscore
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' {
				result += string(r)
			} else {
				result += "_" + string(r)
			}
		} else {
			// Replace dots with underscores
			if r == '.' {
				result += "_"
			} else {
				result += string(r)
			}
		}
	}
	return result
}

// getOrUpdateLabelNames gets the canonical label names for a metric, or updates them
// to include new label names. Returns the union of all label names seen for this metric.
// This method must be called while holding the appropriate mutex for the metric type.
func (a *Adapter) getOrUpdateLabelNames(labelMap *sync.Map, key string, newLabels []string) []string {
	// Try to get existing label names
	if existing, ok := labelMap.Load(key); ok {
		existingLabels := existing.([]string)
		// Compute union of existing and new labels
		labelSet := make(map[string]bool)
		for _, l := range existingLabels {
			labelSet[l] = true
		}
		for _, l := range newLabels {
			labelSet[l] = true
		}

		// Convert back to sorted slice
		union := make([]string, 0, len(labelSet))
		for l := range labelSet {
			union = append(union, l)
		}
		sort.Strings(union)

		// Update if we have new labels
		if len(union) > len(existingLabels) {
			labelMap.Store(key, union)
		}
		return union
	}

	// First time seeing this metric, store the label names
	sorted := make([]string, len(newLabels))
	copy(sorted, newLabels)
	sort.Strings(sorted)
	labelMap.Store(key, sorted)
	return sorted
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

// getLabelsForCanonicalSet creates a Labels map with all canonical label names,
// using values from tags where available. For missing labels, we use "unknown"
// as a sentinel value since Prometheus requires all label values to be non-empty.
// This is a common practice in Prometheus metrics.
// Note: This means metrics with different label combinations will be tracked separately.
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

// getCanonicalLabels gets the canonical label names for a metric, or returns the current labels if not yet set.
func (a *Adapter) getCanonicalLabels(info *metricVecInfo, name string, currentLabels []string) []string {
	key := fmt.Sprintf("%s:%s", info.prefix, name)
	if existing, ok := info.labels.Load(key); ok {
		return existing.([]string)
	}
	return currentLabels
}

// Count implements zstats.Adapter.Count
func (a *Adapter) Count(name string, value float64, tags zstats.Tags) {
	labelNames, _ := a.getLabelNamesAndValues(tags)
	vec := a.getOrCreateCounterVec(name, labelNames)
	canonicalLabels := a.getCanonicalLabels(a.counter, name, labelNames)
	labels := a.getLabelsForCanonicalSet(canonicalLabels, tags)
	vec.With(labels).Add(float64(value))
}

// Gauge implements zstats.Adapter.Gauge
func (a *Adapter) Gauge(name string, value float64, tags zstats.Tags) {
	labelNames, _ := a.getLabelNamesAndValues(tags)
	vec := a.getOrCreateGaugeVec(name, labelNames)
	canonicalLabels := a.getCanonicalLabels(a.gauge, name, labelNames)
	labels := a.getLabelsForCanonicalSet(canonicalLabels, tags)
	vec.With(labels).Set(float64(value))
}

// Timer implements zstats.Adapter.Timer
func (a *Adapter) Timer(name string, cb func(), tags zstats.Tags) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Nanoseconds()
		labelNames, _ := a.getLabelNamesAndValues(tags)
		vec := a.getOrCreateHistogramVec(name, labelNames)
		canonicalLabels := a.getCanonicalLabels(a.hist, name, labelNames)
		labels := a.getLabelsForCanonicalSet(canonicalLabels, tags)
		vec.With(labels).Observe(float64(duration))
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

	go (func() {
		err := server.Serve(listener)
		if err != nil {
			zlog.FromContext(ctx).Errorf("failed to start Prometheus metrics server: %v", err)
		}
	})()

	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
	}()

	return nil
}
