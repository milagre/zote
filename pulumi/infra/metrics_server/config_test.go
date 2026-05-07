package metrics_server_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/infra/metrics_server"
)

func TestConfig_Validate_acceptsEmpty(t *testing.T) {
	t.Parallel()

	c := metrics_server.Config{}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}
