package alloy_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/infra/grafana_stack/alloy"
)

func TestConfig_Validate_acceptsEmpty(t *testing.T) {
	t.Parallel()

	c := alloy.Config{}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}
