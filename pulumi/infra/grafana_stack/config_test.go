package grafana_stack_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/infra/grafana_stack"
	"github.com/milagre/zote/pulumi/infra/grafana_stack/alloy"
	"github.com/milagre/zote/pulumi/infra/grafana_stack/grafana"
	"github.com/milagre/zote/pulumi/infra/grafana_stack/loki"
	"github.com/milagre/zote/pulumi/infra/grafana_stack/mimir"
	"github.com/milagre/zote/pulumi/util/profile"
)

func validLokiProfile() profile.Raw {
	return profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "100m"},
		Mem: profile.RawRange{Min: "64M", Max: "128M"},
	}
}

func TestConfig_Validate_ok(t *testing.T) {
	t.Parallel()

	c := grafana_stack.Config{
		Dashboard: &grafana.Config{},
		Alloy:     &alloy.Config{Version: "1.8.1"},
		Loki: &loki.Config{
			Version:    "7.0.0",
			Monolithic: false,
			Profile:    validLokiProfile(),
			Bucket:     "loki",
		},
		Mimir: &mimir.Config{Monolithic: true, Bucket: "mimir"},
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestConfig_Validate_requiresSubsections(t *testing.T) {
	t.Parallel()

	cases := map[string]grafana_stack.Config{
		"nil dashboard": {
			Dashboard: nil,
			Alloy:     &alloy.Config{},
			Loki:      &loki.Config{Version: "7.0.0", Monolithic: false, Profile: validLokiProfile(), Bucket: "loki"},
			Mimir:     &mimir.Config{Bucket: "m"},
		},
		"nil loki": {
			Dashboard: &grafana.Config{},
			Alloy:     &alloy.Config{},
			Loki:      nil,
			Mimir:     &mimir.Config{Bucket: "m"},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := c.Validate(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
