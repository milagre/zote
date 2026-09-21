package kube_state_metrics

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/infra/grafana_stack"
)

// The pod must opt into the cluster's annotation scrape on the port that serves
// object metrics, and keep its own labels: every series it exports names the
// namespace of the object it describes.
func TestChartValues_podAnnotations(t *testing.T) {
	t.Parallel()

	annotations, ok := chartValues()["podAnnotations"].(pulumi.StringMap)
	if !ok {
		t.Fatalf("podAnnotations missing")
	}

	for key, want := range map[string]string{
		"prometheus.io/scrape":              "true",
		"prometheus.io/port":                "8080",
		grafana_stack.HonorLabelsAnnotation: "true",
	} {
		if got, ok := annotations[key]; !ok || got != pulumi.String(want) {
			t.Errorf("podAnnotations[%q] = %v, want %q", key, got, want)
		}
	}
}
