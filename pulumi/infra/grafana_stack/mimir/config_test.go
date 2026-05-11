package mimir_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/infra/grafana_stack/mimir"
)

func TestConfig_Validate_acceptsEmpty(t *testing.T) {
	t.Parallel()

	var c mimir.Config
	if err := c.Validate(); err == nil {
		t.Errorf("Validate: got nil, want error")
	}
}

func TestConfig_Validate_acceptsBucket(t *testing.T) {
	t.Parallel()

	c := mimir.Config{Bucket: "mimir"}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}
