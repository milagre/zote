package env_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/env"
)

func TestNew_validates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		typ     string
		tier    string
		env     string
		root    string
		prefix  string
		wantErr bool
	}{
		{name: "ok", typ: "local", tier: "dev", env: "username", root: "/tmp", prefix: "p"},
		{name: "missing type", tier: "dev", env: "username", root: "/tmp", wantErr: true},
		{name: "missing tier", typ: "local", env: "username", root: "/tmp", wantErr: true},
		{name: "missing name", typ: "local", tier: "dev", root: "/tmp", wantErr: true},
		{name: "missing root", typ: "local", tier: "dev", env: "username", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := env.New(tc.typ, tc.tier, tc.env, tc.root, tc.prefix)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNew_withRotateSecrets(t *testing.T) {
	t.Parallel()

	e, err := env.New("local", "dev", "username", "/tmp", "", env.WithRotateSecrets("v1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.RotateSecrets != "v1" {
		t.Fatalf("RotateSecrets: got %q want v1", e.RotateSecrets)
	}
}

func TestEnv_derived(t *testing.T) {
	t.Parallel()

	e, err := env.New("local", "dev", "username", "/tmp", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := e.ID(), "dev-username"; got != want {
		t.Errorf("ID: got %q, want %q", got, want)
	}
	if !e.IsDev() {
		t.Error("IsDev: want true")
	}
	if !e.IsLocal() {
		t.Error("IsLocal: want true")
	}
	if got, want := e.LBType(), "NodePort"; got != want {
		t.Errorf("LBType (local): got %q, want %q", got, want)
	}

	prod, _ := env.New("remote", "prod", "us-e1", "/tmp", "")
	if prod.IsDev() {
		t.Error("IsDev (prod): want false")
	}
	if prod.IsLocal() {
		t.Error("IsLocal (remote): want false")
	}
	if got, want := prod.LBType(), "LoadBalancer"; got != want {
		t.Errorf("LBType (remote): got %q, want %q", got, want)
	}
}
