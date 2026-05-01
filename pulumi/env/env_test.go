package env_test

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
)

func TestRandomKeepersDefaultMergesRotateSecrets(t *testing.T) {
	t.Parallel()

	e := env.Env{RotateSecrets: "bump"}
	out := e.RandomKeepers(nil)
	sm, ok := out.(pulumi.StringMap)
	if !ok {
		t.Fatalf("expected pulumi.StringMap, got %T", out)
	}
	if len(sm) != 1 {
		t.Fatalf("expected 1 keeper, got %d", len(sm))
	}
	if _, ok := sm[env.RotateSecretsKeeperKey]; !ok {
		t.Fatalf("missing %q in %#v", env.RotateSecretsKeeperKey, sm)
	}
}

func TestRandomKeepersSupportsRotationFalseOmitsRotateSecrets(t *testing.T) {
	t.Parallel()

	e := env.Env{RotateSecrets: "bump"}
	base := pulumi.StringMap{"k": pulumi.String("v")}
	out := e.RandomKeepers(base, env.SupportsRotation(false))
	sm, ok := out.(pulumi.StringMap)
	if !ok {
		t.Fatalf("expected pulumi.StringMap, got %T", out)
	}
	if len(sm) != 1 {
		t.Fatalf("expected 1 entry, got %d: %#v", len(sm), sm)
	}
	if _, ok := sm[env.RotateSecretsKeeperKey]; ok {
		t.Fatalf("rotateSecrets keeper should not be merged when SupportsRotation is false")
	}
}

func TestRandomKeepersSupportsRotationTrueMergesIntoBase(t *testing.T) {
	t.Parallel()

	e := env.Env{RotateSecrets: "v1"}
	base := pulumi.StringMap{"a": pulumi.String("b")}
	out := e.RandomKeepers(base, env.SupportsRotation(true))
	sm, ok := out.(pulumi.StringMap)
	if !ok {
		t.Fatalf("expected pulumi.StringMap, got %T", out)
	}
	if len(sm) != 2 {
		t.Fatalf("expected 2 entries, got %d: %#v", len(sm), sm)
	}
	if _, ok := sm[env.RotateSecretsKeeperKey]; !ok {
		t.Fatalf("missing rotateSecrets keeper")
	}
}
