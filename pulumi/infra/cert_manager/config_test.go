package cert_manager_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/infra/cert_manager"
)

func TestConfig_Validate_acceptsEmpty(t *testing.T) {
	t.Parallel()

	c := cert_manager.Config{}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}
