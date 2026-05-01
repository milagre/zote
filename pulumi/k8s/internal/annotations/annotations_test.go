package annotations

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func TestSkipAwaitKey(t *testing.T) {
	if SkipAwaitKey != "pulumi.com/skipAwait" {
		t.Fatalf("SkipAwaitKey = %q", SkipAwaitKey)
	}
}

func TestPatchForceKey(t *testing.T) {
	if PatchForceKey != "pulumi.com/patchForce" {
		t.Fatalf("PatchForceKey = %q", PatchForceKey)
	}
}

func TestManagedIsSkipAwaitOnly(t *testing.T) {
	m := Managed()
	if len(m) != 1 {
		t.Fatalf("Managed() len = %d, want 1", len(m))
	}
	v, ok := m[SkipAwaitKey].(pulumi.String)
	if !ok {
		t.Fatalf("Managed()[skipAwait] type %T, want pulumi.String", m[SkipAwaitKey])
	}
	if v != pulumi.String("ready") {
		t.Fatalf("Managed() skipAwait = %q, want %q", v, "ready")
	}
	if _, ok := m[PatchForceKey]; ok {
		t.Fatal("Managed() must not set patch force")
	}
}

func TestManagedWithOverlaysExtra(t *testing.T) {
	m := ManagedWith(pulumi.StringMap{
		"kubernetes.io/ingress.class": pulumi.String("nginx"),
	})
	if len(m) != 2 {
		t.Fatalf("len = %d, want 2", len(m))
	}
	if m[SkipAwaitKey] != pulumi.String("ready") {
		t.Fatalf("skipAwait = %v", m[SkipAwaitKey])
	}
	if m["kubernetes.io/ingress.class"] != pulumi.String("nginx") {
		t.Fatalf("ingress.class = %v", m["kubernetes.io/ingress.class"])
	}
}

func TestManagedWithExtraWinsOnKeyConflict(t *testing.T) {
	m := ManagedWith(pulumi.StringMap{
		SkipAwaitKey: pulumi.String("true"),
	})
	if m[SkipAwaitKey] != pulumi.String("true") {
		t.Fatalf("extra should replace Managed skipAwait, got %v", m[SkipAwaitKey])
	}
}

func TestPatchForce(t *testing.T) {
	m := PatchForce()
	if len(m) != 1 {
		t.Fatalf("PatchForce() len = %d, want 1", len(m))
	}
	if _, ok := m[PatchForceKey]; !ok {
		t.Fatal("PatchForce() missing patch force")
	}
}
