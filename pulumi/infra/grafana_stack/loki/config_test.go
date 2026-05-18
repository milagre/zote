package loki_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/infra/grafana_stack/loki"
	"github.com/milagre/zote/pulumi/util/profile"
)

func validProfile() profile.Raw {
	return profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "100m"},
		Mem: profile.RawRange{Min: "64M", Max: "128M"},
	}
}

func TestConfig_Validate_acceptsLocal(t *testing.T) {
	t.Parallel()

	c := loki.Config{
		Profile: validProfile(),
		Bucket:  "loki",
	}

	if err := c.Validate(); err != nil {
		t.Errorf("Validate: got %v, want nil", err)
	}
}

func TestConfig_Validate_rejectsInvalidProfile(t *testing.T) {
	t.Parallel()

	c := loki.Config{
		Profile: profile.Raw{}, // empty CPU/Mem strings
		Bucket:  "loki",
	}

	err := c.Validate()
	if err == nil {
		t.Errorf("expected profile error, got %v", err)
	}
}
