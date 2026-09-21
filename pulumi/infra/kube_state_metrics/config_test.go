package kube_state_metrics_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/infra/kube_state_metrics"
)

func TestConfig_Validate_acceptsEmpty(t *testing.T) {
	t.Parallel()

	c := kube_state_metrics.Config{}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}
