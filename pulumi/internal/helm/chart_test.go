package helm_test

import (
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/internal/helm"
)

func TestRegisterChartComponent_rejectsNilContext(t *testing.T) {
	t.Parallel()

	spec := helm.ChartSpec{TypeToken: "zote:infra:Test", Chart: "c"}
	comp := &helm.ChartComponent{}
	err := helm.RegisterChartComponent(nil, "ns", "rel", spec, comp)
	if err == nil || !strings.Contains(err.Error(), "context") {
		t.Fatalf("expected context error, got %v", err)
	}
}

func TestRegisterChartComponentNamed_rejectsNilContext(t *testing.T) {
	t.Parallel()

	spec := helm.ChartSpec{TypeToken: "zote:infra:Test", Chart: "c"}
	comp := &helm.ChartComponent{}
	err := helm.RegisterChartComponentNamed(nil, "ns-x", spec, comp)
	if err == nil || !strings.Contains(err.Error(), "context") {
		t.Fatalf("expected context error, got %v", err)
	}
}
