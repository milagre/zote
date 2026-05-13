package body

import (
	"strconv"
)

const (
	// MetricsListenPort is the container listen port for /metrics when workloads opt into Prometheus scraping.
	MetricsListenPort = 9090
)

func metricsAnnotations(port int) map[string]string {
	return map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/path":   "/metrics",
		"prometheus.io/port":   strconv.Itoa(port),
	}
}
