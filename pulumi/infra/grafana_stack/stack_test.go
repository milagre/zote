package grafana_stack

import (
	"fmt"
	"strings"
	"testing"
)

func TestDefaultAlloyRiverTemplate_includesScrapes(t *testing.T) {
	t.Parallel()

	cfg := fmt.Sprintf(defaultAlloyRiverTemplate, "http://mimir/push", "http://loki/push")

	for _, want := range []string{
		"prometheus.operator.podmonitors",
		"prometheus.operator.servicemonitors",
		"discovery.kubernetes",
		"__meta_kubernetes_pod_annotation_prometheus_io_scrape",
		"prometheus.scrape \"pods_annotations\"",
	} {
		if !strings.Contains(cfg, want) {
			t.Errorf("config missing %q", want)
		}
	}
}
