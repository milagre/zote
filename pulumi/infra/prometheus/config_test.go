package prometheus_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/infra/prometheus"
)

func TestConfig_Validate_acceptsEmpty(t *testing.T) {
	t.Parallel()

	c := prometheus.Config{}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}
