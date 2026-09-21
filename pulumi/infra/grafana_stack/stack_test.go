package grafana_stack

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDefaultAlloyRiverTemplate_includesScrapes(t *testing.T) {
	t.Parallel()

	cfg := fmt.Sprintf(defaultAlloyRiverTemplate, "http://mimir/push", 15*time.Second, "http://loki/push")

	for _, want := range []string{
		"prometheus.operator.podmonitors",
		"prometheus.operator.servicemonitors",
		"discovery.kubernetes",
		"__meta_kubernetes_pod_annotation_prometheus_io_scrape",
		"__meta_kubernetes_pod_label_service",
		`target_label  = "namespace"`,
		`target_label  = "name"`,
		`target_label  = "service"`,
		"prometheus.relabel \"zote_metrics\"",
		`regex         = "influxdb;(.+)"`,
		`replacement   = "influxdb_$1"`,
		`forward_to      = [prometheus.relabel.zote_metrics.receiver]`,
		"prometheus.scrape \"pods_annotations\"",
		"loki.source.kubernetes \"pod_logs\"",
		"discovery.kubernetes \"pod_logs\"",
		"loki.write \"default\"",
	} {
		if !strings.Contains(cfg, want) {
			t.Errorf("config missing %q", want)
		}
	}
}

// A pod annotated to honor its own labels is scraped apart from the rest, so
// labels it exports (e.g. the namespace of an object it describes) are not
// overwritten by the labels of the pod that exported them.
func TestDefaultAlloyRiverTemplate_honorLabelsScrape(t *testing.T) {
	t.Parallel()

	cfg := fmt.Sprintf(defaultAlloyRiverTemplate, "http://mimir/push", 15*time.Second, "http://loki/push")

	for _, want := range []string{
		"__meta_kubernetes_pod_annotation_" + strings.NewReplacer(".", "_", "/", "_", "-", "_").Replace(HonorLabelsAnnotation),
		"prometheus.scrape \"pods_annotations_honor_labels\"",
		"honor_labels    = true",
	} {
		if !strings.Contains(cfg, want) {
			t.Errorf("config missing %q", want)
		}
	}
}
