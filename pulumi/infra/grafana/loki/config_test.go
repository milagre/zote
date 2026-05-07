package loki_test

import (
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/infra/grafana/loki"
	"github.com/milagre/zote/pulumi/infra/grafana/loki/internal/digitalocean"
	"github.com/milagre/zote/pulumi/infra/grafana/loki/internal/local"
	"github.com/milagre/zote/pulumi/profile"
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
		Local:   &local.Spec{},
	}

	if err := c.Validate(); err != nil {
		t.Errorf("Validate: got %v, want nil", err)
	}
}

func TestConfig_Validate_acceptsCloudDigitalOcean(t *testing.T) {
	t.Parallel()

	c := loki.Config{
		Profile: validProfile(),
		Cloud: &loki.Cloud{
			DigitalOcean: &digitalocean.Spec{},
		},
	}

	if err := c.Validate(); err != nil {
		t.Errorf("Validate: got %v, want nil", err)
	}
}

func TestConfig_Validate_rejectsBothBackends(t *testing.T) {
	t.Parallel()

	c := loki.Config{
		Profile: validProfile(),
		Cloud:   &loki.Cloud{DigitalOcean: &digitalocean.Spec{}},
		Local:   &local.Spec{},
	}

	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected mutually-exclusive error, got %v", err)
	}
}

func TestConfig_Validate_rejectsNeitherBackend(t *testing.T) {
	t.Parallel()

	c := loki.Config{
		Profile: validProfile(),
	}

	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "storage backend is required") {
		t.Errorf("expected backend-required error, got %v", err)
	}
}

func TestConfig_Validate_rejectsEmptyCloud(t *testing.T) {
	t.Parallel()

	c := loki.Config{
		Profile: validProfile(),
		Cloud:   &loki.Cloud{},
	}

	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "at least one provider") {
		t.Errorf("expected provider-required error, got %v", err)
	}
}

func TestConfig_Validate_rejectsInvalidProfile(t *testing.T) {
	t.Parallel()

	c := loki.Config{
		Profile: profile.Raw{}, // empty CPU/Mem strings
		Local:   &local.Spec{},
	}

	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "profile") {
		t.Errorf("expected profile error, got %v", err)
	}
}
