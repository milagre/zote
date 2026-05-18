package container

import "strconv"

// PrometheusPodAnnotations are pod template annotations for Alloy's annotation-based scrape.
func PrometheusPodAnnotations() map[string]string {
	return map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/path":   "/metrics",
		"prometheus.io/port":   strconv.Itoa(portPrometheus),
	}
}
