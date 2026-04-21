package mysql

import (
	"strings"
	"testing"

	"github.com/milagre/zote/pulumi/env"
)

func validEnv(t *testing.T) env.Env {
	t.Helper()
	e, err := env.New("prod", "prod", "mars", "/home/mars", "MARS")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}
	return e
}

func baseArgs(t *testing.T) *Args {
	t.Helper()
	return &Args{
		Env:       validEnv(t),
		Namespace: "apps",
		Name:      "orders",
		Version:   "8.0",
		Database:  "orders",
		Username:  "orders_rw",
	}
}

func TestArgsValidate_requiresBackendSelection(t *testing.T) {
	t.Parallel()

	a := baseArgs(t)
	err := a.validate()
	if err == nil {
		t.Fatalf("expected error for missing backend, got nil")
	}
	if !strings.Contains(err.Error(), "backend is required") {
		t.Errorf("expected missing-backend error, got %v", err)
	}
}

func TestArgsValidate_rejectsBothBackends(t *testing.T) {
	t.Parallel()

	a := baseArgs(t)
	a.Container = &ContainerArgs{}
	a.DigitalOcean = &DigitalOceanArgs{}
	err := a.validate()
	if err == nil {
		t.Fatalf("expected error for dual backends, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected mutual-exclusion error, got %v", err)
	}
}

func TestArgsValidate_acceptsContainerBackend(t *testing.T) {
	t.Parallel()

	a := baseArgs(t)
	a.Container = &ContainerArgs{}
	if err := a.validate(); err != nil {
		t.Errorf("expected valid container args, got %v", err)
	}
}

func TestArgsValidate_acceptsDigitaloceanBackend(t *testing.T) {
	t.Parallel()

	a := baseArgs(t)
	a.DigitalOcean = &DigitalOceanArgs{}
	if err := a.validate(); err != nil {
		t.Errorf("expected valid DO args at the facade layer, got %v", err)
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
		{"no Version", func(a *Args) { a.Version = "" }, "Version is required"},
		{"no Database", func(a *Args) { a.Database = "" }, "Database is required"},
		{"no Username", func(a *Args) { a.Username = "" }, "Username is required"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			a := baseArgs(t)
			a.Container = &ContainerArgs{}
			tc.mut(a)
			err := a.validate()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestArgsValidate_wrapsEnvValidationFailure(t *testing.T) {
	t.Parallel()

	a := baseArgs(t)
	a.Container = &ContainerArgs{}
	a.Env = env.Env{}
	err := a.validate()
	if err == nil {
		t.Fatalf("expected env validation error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid env") {
		t.Errorf("expected wrapped env error, got %v", err)
	}
}
