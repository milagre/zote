package cloudflare_tunnel_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/infra/cloudflare_tunnel"
)

func TestConfig_Validate_acceptsEmpty(t *testing.T) {
	t.Parallel()

	c := cloudflare_tunnel.Config{}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}
