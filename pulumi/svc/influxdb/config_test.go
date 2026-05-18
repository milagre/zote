package influxdb_test

import (
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/svc/influxdb"
	"github.com/milagre/zote/pulumi/svc/influxdb/internal/container"
	"github.com/milagre/zote/pulumi/util/profile"
)

func rawProf() profile.Raw {
	return profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "100m"},
		Mem: profile.RawRange{Min: "64M", Max: "128M"},
	}
}

func validInfluxConfig() influxdb.Config {
	return influxdb.Config{
		Version: "2.7",
		Container: &container.Spec{
			Profile: rawProf(),
		},
	}
}

func TestConfig_Validate_acceptsMinimal(t *testing.T) {
	t.Parallel()

	c := validInfluxConfig()
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestConfig_Validate_rejectsMissingVersion(t *testing.T) {
	t.Parallel()

	c := validInfluxConfig()
	c.Version = ""

	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "version") {
		t.Errorf("expected version error, got %v", err)
	}
}

func TestOrganizationOrDefault(t *testing.T) {
	t.Parallel()

	c := validInfluxConfig()
	if got, want := c.OrganizationOrDefault(), "influxdb"; got != want {
		t.Errorf("OrganizationOrDefault: got %q, want %q", got, want)
	}
	c.Organization = "custom"
	if got, want := c.OrganizationOrDefault(), "custom"; got != want {
		t.Errorf("OrganizationOrDefault: got %q, want %q", got, want)
	}
}
