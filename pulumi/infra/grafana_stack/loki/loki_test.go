package loki_test

import (
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra/grafana_stack/loki"
	"github.com/milagre/zote/pulumi/infra/objectstorage"
	"github.com/milagre/zote/pulumi/profile"
)

func lokiLocalConfig() loki.Config {
	return loki.Config{
		Monolithic: true,
		Profile:    validProfile(),
		Bucket:     "loki",
	}
}

func objectStorage() objectstorage.ObjectStorage {
	// Validation uses Buckets only; Resource is wired in real stacks.
	return objectstorage.ObjectStorage{Buckets: []string{"loki"}}
}

func localTierEnv(t *testing.T) env.Env {
	t.Helper()

	e, err := env.New("zote", "local", "dev", "user", "/tmp", "ZOTE")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	return e
}

func remoteEnv(t *testing.T) env.Env {
	t.Helper()

	e, err := env.New("zote", "remote", "prod", "prod", "/tmp", "ZOTE")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	return e
}

func TestArgs_validate_rejectsEmptyNamespace(t *testing.T) {
	t.Parallel()

	args := &loki.Args{
		Env:           localTierEnv(t),
		Config:        lokiLocalConfig(),
		ObjectStorage: objectStorage(),
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
			Profile: profile.Raw{}, // invalid CPU/Mem
		},
		ObjectStorage: objectStorage(),
	}

	if _, err := loki.New(nil, "loki", args); err == nil ||
		!strings.Contains(err.Error(), "profile") {
		t.Errorf("expected config error, got %v", err)
	}
}

func TestArgs_validate_rejectsMissingObjectStorage(t *testing.T) {
	t.Parallel()

	args := &loki.Args{
		Env:       remoteEnv(t),
		Namespace: "infra",
		Config:    lokiLocalConfig(),
	}

	if _, err := loki.New(nil, "loki", args); err == nil ||
		!strings.Contains(err.Error(), "does not contain bucket") {
		t.Errorf("expected objectstorage-required error, got %v", err)
	}
}

func TestArgs_validate_rejectsBucketNotInObjectStorage(t *testing.T) {
	t.Parallel()

	args := &loki.Args{
		Env:           remoteEnv(t),
		Namespace:     "infra",
		Config:        lokiLocalConfig(),
		ObjectStorage: objectstorage.ObjectStorage{Buckets: []string{"other"}},
	}

	if _, err := loki.New(nil, "loki", args); err == nil ||
		!strings.Contains(err.Error(), "does not contain bucket") {
		t.Errorf("expected bucket-missing error, got %v", err)
	}
}
