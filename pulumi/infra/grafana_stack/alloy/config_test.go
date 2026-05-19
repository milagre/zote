package alloy_test

import (
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/infra/grafana_stack/alloy"
	"github.com/milagre/zote/pulumi/util/profile"
)

func TestConfig_Validate_acceptsEmpty(t *testing.T) {
	t.Parallel()

	c := alloy.Config{}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestConfig_Validate_profile(t *testing.T) {
	t.Parallel()

	valid := profile.Raw{
		CPU: profile.RawRange{Min: "50m", Max: "100m"},
		Mem: profile.RawRange{Min: "64M", Max: "128M"},
	}
	if err := (&alloy.Config{Profile: valid}).Validate(); err != nil {
		t.Fatalf("valid profile: %v", err)
	}

	err := (&alloy.Config{Profile: profile.Raw{CPU: profile.RawRange{Min: "50m"}}}).Validate()
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("partial profile: got %v", err)
	}
}
