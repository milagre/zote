package loki_test

import (
	"strings"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/cloud"
	docloud "github.com/milagre/zote/pulumi/cloud/digitalocean"
	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra/grafana/loki"
	"github.com/milagre/zote/pulumi/infra/grafana/loki/internal/digitalocean"
	"github.com/milagre/zote/pulumi/infra/grafana/loki/internal/local"
)

func validCloud() cloud.Cloud {
	return cloud.Cloud{
		DigitalOcean: docloud.New(pulumi.String("vpc-1"), pulumi.String("proj-1")),
	}
}

func lokiLocalConfig() loki.Config {
	return loki.Config{
		Profile: validProfile(),
		Local:   &local.Spec{},
	}
}

func cloudConfig() loki.Config {
	return loki.Config{
		Profile: validProfile(),
		Cloud: &loki.Cloud{
			DigitalOcean: &digitalocean.Spec{},
		},
	}
}

func localTierEnv(t *testing.T) env.Env {
	t.Helper()

	e, err := env.New("local", "dev", "user", "/tmp", "ZOTE")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	return e
}

func remoteEnv(t *testing.T) env.Env {
	t.Helper()

	e, err := env.New("remote", "prod", "prod", "/tmp", "ZOTE")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	return e
}

func TestArgs_validate_rejectsEmptyNamespace(t *testing.T) {
	t.Parallel()

	args := &loki.Args{
		Env:    localTierEnv(t),
		Config: lokiLocalConfig(),
	}

	if _, err := loki.New(nil, "loki", args); err == nil ||
		!strings.Contains(err.Error(), "Namespace is required") {
		t.Errorf("expected Namespace error, got %v", err)
	}
}

func TestArgs_validate_propagatesConfigErrors(t *testing.T) {
	t.Parallel()

	args := &loki.Args{
		Env:       localTierEnv(t),
		Namespace: "infra",
		Config: loki.Config{
			Profile: validProfile(),
			// no backend
		},
	}

	if _, err := loki.New(nil, "loki", args); err == nil ||
		!strings.Contains(err.Error(), "storage backend is required") {
		t.Errorf("expected backend-required error, got %v", err)
	}
}

func TestArgs_validate_rejectsLocalWithCloud(t *testing.T) {
	t.Parallel()

	args := &loki.Args{
		Env:       localTierEnv(t),
		Namespace: "infra",
		Config:    cloudConfig(),
		Cloud:     validCloud(),
	}

	if _, err := loki.New(nil, "loki", args); err == nil ||
		!strings.Contains(err.Error(), "local environments require the local backend") {
		t.Errorf("expected local-requires-local-backend error, got %v", err)
	}
}

func TestArgs_validate_rejectsRemoteWithLocal(t *testing.T) {
	t.Parallel()

	args := &loki.Args{
		Env:       remoteEnv(t),
		Namespace: "infra",
		Config:    lokiLocalConfig(),
	}

	if _, err := loki.New(nil, "loki", args); err == nil ||
		!strings.Contains(err.Error(), "non-local environments require a cloud backend") {
		t.Errorf("expected remote-requires-cloud error, got %v", err)
	}
}

func TestArgs_validate_rejectsCloudWithoutDigitalOceanHandle(t *testing.T) {
	t.Parallel()

	args := &loki.Args{
		Env:       remoteEnv(t),
		Namespace: "infra",
		Config:    cloudConfig(),
		// Cloud is the zero value: no providers populated.
	}

	if _, err := loki.New(nil, "loki", args); err == nil ||
		!strings.Contains(err.Error(), "Cloud.DigitalOcean is required") {
		t.Errorf("expected handle-required error, got %v", err)
	}
}
