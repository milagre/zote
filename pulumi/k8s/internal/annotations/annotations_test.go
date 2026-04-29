package annotations

import "testing"

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
	if _, ok := m[SkipAwaitKey]; !ok {
		t.Fatal("Managed() missing skip await")
	}
	if _, ok := m[PatchForceKey]; ok {
		t.Fatal("Managed() must not set patch force")
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
