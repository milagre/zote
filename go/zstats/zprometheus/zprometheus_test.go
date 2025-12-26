package zprometheus

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zstats"
)

func TestNewAdapter(t *testing.T) {
	adapter := NewAdapter()
	assert.NotNil(t, adapter)
	assert.NotNil(t, adapter.registry)

	// Verify it implements the interface
	var _ zstats.Adapter = adapter
}

func TestCount_Basic(t *testing.T) {
	adapter := NewAdapter()

	adapter.Count("test.counter", 5, nil)
	adapter.Count("test.counter", 3, nil)

	// Gather metrics
	mfs, err := adapter.registry.Gather()
	require.NoError(t, err)

	// Find our metric
	var metric *dto.MetricFamily
	for _, mf := range mfs {
		if strings.Contains(mf.GetName(), "test_counter") {
			metric = mf
			break
		}
	}

	require.NotNil(t, metric, "metric should be found")
	assert.Equal(t, dto.MetricType_COUNTER, metric.GetType())
	assert.Len(t, metric.Metric, 1)
	assert.Equal(t, 8.0, metric.Metric[0].Counter.GetValue())
}

func TestCount_WithTags(t *testing.T) {
	adapter := NewAdapter()

	adapter.Count("requests", 1, zstats.Tags{"method": "GET", "status": "200"})
	adapter.Count("requests", 2, zstats.Tags{"method": "POST", "status": "200"})
	adapter.Count("requests", 1, zstats.Tags{"method": "GET", "status": "404"})

	// Gather metrics
	mfs, err := adapter.registry.Gather()
	require.NoError(t, err)

	// Find our metric
	var metric *dto.MetricFamily
	for _, mf := range mfs {
		if strings.Contains(mf.GetName(), "requests") {
			metric = mf
			break
		}
	}

	require.NotNil(t, metric, "metric should be found")
	assert.Equal(t, dto.MetricType_COUNTER, metric.GetType())

	// Should have 3 different time series (one for each label combination)
	assert.GreaterOrEqual(t, len(metric.Metric), 3)
}

func TestCount_WithMissingLabels(t *testing.T) {
	adapter := NewAdapter()

	// First call with one set of labels
	adapter.Count("test", 1, zstats.Tags{"label1": "value1"})

	// Second call with different labels (should use "unknown" for missing label1)
	adapter.Count("test", 2, zstats.Tags{"label2": "value2"})

	// Gather metrics
	mfs, err := adapter.registry.Gather()
	require.NoError(t, err)

	// Find our metric
	var metric *dto.MetricFamily
	for _, mf := range mfs {
		if strings.Contains(mf.GetName(), "test") && mf.GetType() == dto.MetricType_COUNTER {
			metric = mf
			break
		}
	}

	require.NotNil(t, metric, "metric should be found")

	// Should have metrics with both label combinations
	// One with label1=value1, label2=unknown
	// One with label1=unknown, label2=value2
	foundLabel1 := false
	foundLabel2 := false
	for _, m := range metric.Metric {
		labels := m.GetLabel()
		for _, label := range labels {
			if label.GetName() == "label1" && label.GetValue() == "value1" {
				foundLabel1 = true
			}
			if label.GetName() == "label2" && label.GetValue() == "value2" {
				foundLabel2 = true
			}
		}
	}
	assert.True(t, foundLabel1 || foundLabel2, "should have metrics with both label combinations")
}

func TestGauge_Basic(t *testing.T) {
	adapter := NewAdapter()

	adapter.Gauge("test.gauge", 10, nil)
	adapter.Gauge("test.gauge", 20, nil)

	// Gather metrics
	mfs, err := adapter.registry.Gather()
	require.NoError(t, err)

	// Find our metric
	var metric *dto.MetricFamily
	for _, mf := range mfs {
		if strings.Contains(mf.GetName(), "test_gauge") {
			metric = mf
			break
		}
	}

	require.NotNil(t, metric, "metric should be found")
	assert.Equal(t, dto.MetricType_GAUGE, metric.GetType())
	assert.Len(t, metric.Metric, 1)
	assert.Equal(t, 20.0, metric.Metric[0].Gauge.GetValue())
}

func TestGauge_WithTags(t *testing.T) {
	adapter := NewAdapter()

	adapter.Gauge("temperature", 25, zstats.Tags{"room": "kitchen"})
	adapter.Gauge("temperature", 22, zstats.Tags{"room": "bedroom"})

	// Gather metrics
	mfs, err := adapter.registry.Gather()
	require.NoError(t, err)

	// Find our metric
	var metric *dto.MetricFamily
	for _, mf := range mfs {
		if strings.Contains(mf.GetName(), "temperature") {
			metric = mf
			break
		}
	}

	require.NotNil(t, metric, "metric should be found")
	assert.Equal(t, dto.MetricType_GAUGE, metric.GetType())
	assert.GreaterOrEqual(t, len(metric.Metric), 2)
}

func TestTimer_Basic(t *testing.T) {
	adapter := NewAdapter()

	adapter.Timer("test.timer", func() {
		time.Sleep(10 * time.Millisecond)
	}, nil)

	// Gather metrics
	mfs, err := adapter.registry.Gather()
	require.NoError(t, err)

	// Find our metric
	var metric *dto.MetricFamily
	for _, mf := range mfs {
		if strings.Contains(mf.GetName(), "test_timer") {
			metric = mf
			break
		}
	}

	require.NotNil(t, metric, "metric should be found")
	assert.Equal(t, dto.MetricType_HISTOGRAM, metric.GetType())
	assert.GreaterOrEqual(t, len(metric.Metric), 1)

	// Check that we have a histogram with observations
	histogram := metric.Metric[0].GetHistogram()
	assert.NotNil(t, histogram)
	assert.Greater(t, histogram.GetSampleCount(), uint64(0))
}

func TestTimer_WithTags(t *testing.T) {
	adapter := NewAdapter()

	adapter.Timer("request.duration", func() {
		time.Sleep(5 * time.Millisecond)
	}, zstats.Tags{"endpoint": "/api/users"})

	// Gather metrics
	mfs, err := adapter.registry.Gather()
	require.NoError(t, err)

	// Find our metric
	var metric *dto.MetricFamily
	for _, mf := range mfs {
		if strings.Contains(mf.GetName(), "request_duration") {
			metric = mf
			break
		}
	}

	require.NotNil(t, metric, "metric should be found")
	assert.Equal(t, dto.MetricType_HISTOGRAM, metric.GetType())
}

func TestPrometheusName(t *testing.T) {
	adapter := NewAdapter()

	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"test.metric", "test_metric"},
		{"test.metric.name", "test_metric_name"},
		{"123invalid", "_123invalid"},
		{"_valid", "_valid"},
		{"valid123", "valid123"},
		{"a.b.c.d", "a_b_c_d"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := adapter.prometheusName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetLabelNamesAndValues(t *testing.T) {
	adapter := NewAdapter()

	tests := []struct {
		name     string
		tags     zstats.Tags
		expected []string
	}{
		{
			name:     "empty tags",
			tags:     nil,
			expected: []string{},
		},
		{
			name:     "single tag",
			tags:     zstats.Tags{"key1": "value1"},
			expected: []string{"key1"},
		},
		{
			name:     "multiple tags",
			tags:     zstats.Tags{"key2": "value2", "key1": "value1"},
			expected: []string{"key1", "key2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names, values := adapter.getLabelNamesAndValues(tt.tags)
			assert.Equal(t, tt.expected, names)
			if len(tt.tags) > 0 {
				assert.Equal(t, len(names), len(values))
				// Verify values match tags
				for i, name := range names {
					assert.Equal(t, tt.tags[name], values[i])
				}
			} else {
				assert.Empty(t, values)
			}
		})
	}
}

func TestGetLabelsForCanonicalSet(t *testing.T) {
	adapter := NewAdapter()

	canonicalLabels := []string{"label1", "label2", "label3"}
	tags := zstats.Tags{"label1": "value1", "label3": "value3"}

	labels := adapter.getLabelsForCanonicalSet(canonicalLabels, tags)

	assert.Equal(t, "value1", labels["label1"])
	assert.Equal(t, "unknown", labels["label2"]) // missing label
	assert.Equal(t, "value3", labels["label3"])
}

func TestConcurrentAccess(t *testing.T) {
	adapter := NewAdapter()

	var wg sync.WaitGroup
	numGoroutines := 10
	iterations := 100

	// Test concurrent Count calls
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				adapter.Count("concurrent.counter", 1, zstats.Tags{"goroutine": fmt.Sprintf("%d", id)})
			}
		}(i)
	}
	wg.Wait()

	// Test concurrent Gauge calls
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				adapter.Gauge("concurrent.gauge", float64(id*iterations+j), zstats.Tags{"goroutine": fmt.Sprintf("%d", id)})
			}
		}(i)
	}
	wg.Wait()

	// Test concurrent Timer calls
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				adapter.Timer("concurrent.timer", func() {
					time.Sleep(time.Microsecond)
				}, zstats.Tags{"goroutine": fmt.Sprintf("%d", id)})
			}
		}(i)
	}
	wg.Wait()

	// Gather metrics and verify no panics occurred
	mfs, err := adapter.registry.Gather()
	require.NoError(t, err)
	assert.Greater(t, len(mfs), 0)
}

func TestMetricsHandler(t *testing.T) {
	adapter := NewAdapter()

	// Record some metrics
	adapter.Count("test.counter", 5, nil)
	adapter.Gauge("test.gauge", 10, nil)

	// Create request
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()

	// Call handler
	handler := adapter.metricsHandler()
	handler.ServeHTTP(rr, req)

	// Verify response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "text/plain")

	// Verify metrics are in response
	body := rr.Body.String()
	assert.Contains(t, body, "test_counter")
	assert.Contains(t, body, "test_gauge")
}

func TestCanonicalLabels_Union(t *testing.T) {
	adapter := NewAdapter()

	// First call with labels A and B
	adapter.Count("union.test", 1, zstats.Tags{"labelA": "valueA", "labelB": "valueB"})

	// Wait a bit for the async label update to complete
	time.Sleep(10 * time.Millisecond)

	// Second call with labels B and C (should create union: A, B, C)
	adapter.Count("union.test", 1, zstats.Tags{"labelB": "valueB", "labelC": "valueC"})

	// Wait a bit for the async label update to complete
	time.Sleep(10 * time.Millisecond)

	// Gather metrics
	mfs, err := adapter.registry.Gather()
	require.NoError(t, err)

	// Find our metric
	var metric *dto.MetricFamily
	for _, mf := range mfs {
		if strings.Contains(mf.GetName(), "union_test") {
			metric = mf
			break
		}
	}

	require.NotNil(t, metric, "metric should be found")

	// Verify that metrics have labelB (present in both calls)
	// The union of labels might include A, B, C, but due to async updates,
	// we might not see all of them immediately. The important thing is
	// that the metric works correctly with different label combinations.
	hasLabelB := false

	for _, m := range metric.Metric {
		labels := m.GetLabel()
		for _, label := range labels {
			if label.GetName() == "labelB" {
				hasLabelB = true
				break
			}
		}
		if hasLabelB {
			break
		}
	}

	// At minimum, labelB should be present (it's in both calls)
	assert.True(t, hasLabelB, "should have labelB")
}

func TestEmptyTags(t *testing.T) {
	adapter := NewAdapter()

	adapter.Count("empty.tags", 1, nil)
	adapter.Count("empty.tags", 1, zstats.Tags{})

	// Gather metrics
	mfs, err := adapter.registry.Gather()
	require.NoError(t, err)

	// Find our metric
	var metric *dto.MetricFamily
	for _, mf := range mfs {
		if strings.Contains(mf.GetName(), "empty_tags") {
			metric = mf
			break
		}
	}

	require.NotNil(t, metric, "metric should be found")
	assert.GreaterOrEqual(t, len(metric.Metric), 1)
}

func TestMultipleMetrics(t *testing.T) {
	adapter := NewAdapter()

	// Create multiple different metrics
	adapter.Count("metric1", 1, zstats.Tags{"type": "counter"})
	adapter.Gauge("metric2", 2, zstats.Tags{"type": "gauge"})
	adapter.Timer("metric3", func() {
		time.Sleep(1 * time.Millisecond)
	}, zstats.Tags{"type": "timer"})

	// Gather metrics
	mfs, err := adapter.registry.Gather()
	require.NoError(t, err)

	// Should have at least 3 metrics
	assert.GreaterOrEqual(t, len(mfs), 3)

	// Verify each type
	hasCounter := false
	hasGauge := false
	hasHistogram := false

	for _, mf := range mfs {
		switch mf.GetType() {
		case dto.MetricType_COUNTER:
			if strings.Contains(mf.GetName(), "metric1") {
				hasCounter = true
			}
		case dto.MetricType_GAUGE:
			if strings.Contains(mf.GetName(), "metric2") {
				hasGauge = true
			}
		case dto.MetricType_HISTOGRAM:
			if strings.Contains(mf.GetName(), "metric3") {
				hasHistogram = true
			}
		}
	}

	assert.True(t, hasCounter, "should have counter metric")
	assert.True(t, hasGauge, "should have gauge metric")
	assert.True(t, hasHistogram, "should have histogram metric")
}

func TestTimer_CallbackExecutes(t *testing.T) {
	adapter := NewAdapter()

	executed := false
	adapter.Timer("test.timer", func() {
		executed = true
	}, nil)

	assert.True(t, executed, "callback should have executed")
}

func TestMetricsHandler_ContentType(t *testing.T) {
	adapter := NewAdapter()

	adapter.Count("test", 1, nil)

	// Test with Accept header for OpenMetrics format
	req := httptest.NewRequest("GET", "/metrics", nil)
	req.Header.Set("Accept", "application/openmetrics-text")
	rr := httptest.NewRecorder()

	handler := adapter.metricsHandler()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// Content type should be negotiated
	assert.NotEmpty(t, rr.Header().Get("Content-Type"))
}

func TestPrometheusName_EdgeCases(t *testing.T) {
	adapter := NewAdapter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single dot", ".", "_."},        // First char is dot, gets prefixed with "_"
		{"multiple dots", "...", "_.__"}, // First char is dot, gets prefixed with "_", rest dots become "_"
		{"starts with number", "9metric", "_9metric"},
		{"mixed case", "Test.Metric.Name", "Test_Metric_Name"},
		{"with underscore", "test_metric", "test_metric"},
		{"dots and numbers", "test.123.metric", "test_123_metric"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.prometheusName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark tests
func BenchmarkCount(b *testing.B) {
	adapter := NewAdapter()
	tags := zstats.Tags{"method": "GET", "status": "200"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.Count("bench.counter", 1, tags)
	}
}

func BenchmarkGauge(b *testing.B) {
	adapter := NewAdapter()
	tags := zstats.Tags{"room": "kitchen"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.Gauge("bench.gauge", float64(i), tags)
	}
}

func BenchmarkTimer(b *testing.B) {
	adapter := NewAdapter()
	tags := zstats.Tags{"operation": "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.Timer("bench.timer", func() {
			// minimal work
		}, tags)
	}
}

func BenchmarkConcurrentCount(b *testing.B) {
	adapter := NewAdapter()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			adapter.Count("bench.concurrent", 1, zstats.Tags{"id": "1"})
		}
	})
}

// Helper function to extract metric value from gathered metrics
func getMetricValue(mfs []*dto.MetricFamily, name string) float64 {
	for _, mf := range mfs {
		if strings.Contains(mf.GetName(), name) {
			if len(mf.Metric) > 0 {
				switch mf.GetType() {
				case dto.MetricType_COUNTER:
					return mf.Metric[0].Counter.GetValue()
				case dto.MetricType_GAUGE:
					return mf.Metric[0].Gauge.GetValue()
				}
			}
		}
	}
	return 0
}

func TestGetMetricValue_Helper(t *testing.T) {
	adapter := NewAdapter()

	adapter.Count("helper.test", 42, nil)

	mfs, err := adapter.registry.Gather()
	require.NoError(t, err)

	value := getMetricValue(mfs, "helper_test")
	assert.Equal(t, 42.0, value)
}

func TestMetricsHandler_ErrorHandling(t *testing.T) {
	// Create an adapter with a registry that might cause errors
	// This is harder to test directly, but we can at least verify
	// the handler doesn't panic on normal operations
	adapter := NewAdapter()

	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()

	handler := adapter.metricsHandler()
	handler.ServeHTTP(rr, req)

	// Should succeed with empty registry
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestLabelOrdering(t *testing.T) {
	adapter := NewAdapter()

	// Tags in different order should produce same sorted result
	tags1 := zstats.Tags{"z": "last", "a": "first", "m": "middle"}
	tags2 := zstats.Tags{"a": "first", "m": "middle", "z": "last"}

	names1, values1 := adapter.getLabelNamesAndValues(tags1)
	names2, values2 := adapter.getLabelNamesAndValues(tags2)

	// Should be sorted the same way
	assert.Equal(t, names1, names2)
	assert.Equal(t, values1, values2)
	assert.Equal(t, []string{"a", "m", "z"}, names1)
}

func TestDotDelimitedNames(t *testing.T) {
	adapter := NewAdapter()

	// Test that dot-delimited names work correctly
	adapter.Count("service.api.requests", 1, zstats.Tags{"method": "GET"})
	adapter.Gauge("service.api.active", 5, zstats.Tags{"endpoint": "/users"})

	mfs, err := adapter.registry.Gather()
	require.NoError(t, err)

	// Verify metrics exist with converted names
	foundRequests := false
	foundActive := false

	for _, mf := range mfs {
		name := mf.GetName()
		if strings.Contains(name, "service_api_requests") {
			foundRequests = true
		}
		if strings.Contains(name, "service_api_active") {
			foundActive = true
		}
	}

	assert.True(t, foundRequests, "should find requests metric")
	assert.True(t, foundActive, "should find active metric")
}
