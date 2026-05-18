package nginx_ingress_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/infra/nginx_ingress"
	"github.com/milagre/zote/pulumi/util/profile"
)

func rawProf() profile.Raw {
	return profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "100m"},
		Mem: profile.RawRange{Min: "64M", Max: "128M"},
	}
}

func TestConfig_Validate_acceptsMinimal(t *testing.T) {
	t.Parallel()

	c := nginx_ingress.Config{Profile: rawProf()}
	if err := c.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}
