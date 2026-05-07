package mysql

import (
	"strings"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/cloud"
	docloud "github.com/milagre/zote/pulumi/cloud/digitalocean"
	"github.com/milagre/zote/pulumi/database/mysql/internal/container"
	"github.com/milagre/zote/pulumi/database/mysql/internal/digitalocean"
	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/profile"
)

func validEnv(t *testing.T) env.Env {
	t.Helper()
	e, err := env.New("prod", "prod", "mars", "/home/mars", "MARS")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}
	return e
}

func validRawProfile() profile.Raw {
	return profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "100m"},
		Mem: profile.RawRange{Min: "64M", Max: "128M"},
	}
}

func validContainerConfig() *container.Spec {
	return &container.Spec{
		Primary: container.Tier{Profile: validRawProfile()},
		Replica: container.Tier{Profile: validRawProfile()},
	}
}

func validCloudConfig() *Cloud {
	return &Cloud{
		DigitalOcean: &digitalocean.Spec{
			Primary: digitalocean.Primary{Class: "db-s-1vcpu-1gb"},
		},
	}
}

func validCloud() cloud.Cloud {
	return cloud.Cloud{
		DigitalOcean: docloud.New(pulumi.String("vpc-1"), pulumi.String("proj-1")),
	}
}

func baseArgs(t *testing.T) *Args {
	t.Helper()
	return &Args{
		Env:       validEnv(t),
		Namespace: "apps",
		Name:      "orders",
		Database:  "orders",
		Username:  "orders_rw",
	}
}

func TestArgsValidate_requiresCoreFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		mut  func(a *Args)
		want string
	}{
		{"no Name", func(a *Args) { a.Name = "" }, "Name is required"},
		{"no Namespace", func(a *Args) { a.Namespace = "" }, "Namespace is required"},
		{"no Database", func(a *Args) { a.Database = "" }, "Database is required"},
		{"no Username", func(a *Args) { a.Username = "" }, "Username is required"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			a := baseArgs(t)
			a.Config = Config{Version: "8", Container: validContainerConfig()}
			tc.mut(a)
			err := a.validate()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestArgsValidate_propagatesConfigErrors(t *testing.T) {
	t.Parallel()

	a := baseArgs(t)
	// Config has neither backend, so Config.Validate fails.
	a.Config = Config{Version: "8"}

	if err := a.validate(); err == nil || !strings.Contains(err.Error(), "backend is required") {
		t.Errorf("expected backend-required error, got %v", err)
	}
}

func TestArgsValidate_wrapsEnvValidationFailure(t *testing.T) {
	t.Parallel()

	a := baseArgs(t)
	a.Config = Config{Version: "8", Container: validContainerConfig()}
	a.Env = env.Env{}

	if err := a.validate(); err == nil || !strings.Contains(err.Error(), "invalid env") {
		t.Errorf("expected wrapped env error, got %v", err)
	}
}

func TestArgsValidate_acceptsContainerBackend(t *testing.T) {
	t.Parallel()

	a := baseArgs(t)
	a.Config = Config{Version: "8", Container: validContainerConfig()}
	if err := a.validate(); err != nil {
		t.Errorf("expected valid container args, got %v", err)
	}
}

func TestArgsValidate_acceptsDigitaloceanBackendWithCloud(t *testing.T) {
	t.Parallel()

	a := baseArgs(t)
	a.Config = Config{Version: "8", Cloud: validCloudConfig()}
	a.Cloud = validCloud()

	if err := a.validate(); err != nil {
		t.Errorf("expected valid DO args, got %v", err)
	}
}

func TestArgsValidate_rejectsCloudWithoutDigitalOceanHandle(t *testing.T) {
	t.Parallel()

	a := baseArgs(t)
	a.Config = Config{Version: "8", Cloud: validCloudConfig()}
	// Cloud is the zero value: DigitalOcean handle nil.

	if err := a.validate(); err == nil || !strings.Contains(err.Error(), "Cloud.DigitalOcean is required") {
		t.Errorf("expected handle-required error, got %v", err)
	}
}
