package digitalocean_test

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/cloud/digitalocean"
	dbdo "github.com/milagre/zote/pulumi/database/digitalocean"
	osdo "github.com/milagre/zote/pulumi/infra/objectstorage/digitalocean"
)

var _ dbdo.Cloud = (*digitalocean.DatabaseCloud)(nil)
var _ osdo.Cloud = (*digitalocean.ObjectStorageCloud)(nil)

func newCloud(vpc, proj string) *digitalocean.Cloud {
	return digitalocean.New(pulumi.String(vpc), pulumi.String(proj))
}

func TestCloud_publicLoadBalancerAnnotations(t *testing.T) {
	t.Parallel()

	got := newCloud("vpc", "proj").PublicLoadBalancerAnnotations()

	if got["service.beta.kubernetes.io/do-loadbalancer-tls-ports"] != "443" {
		t.Errorf("missing DO TLS ports annotation: %+v", got)
	}
	if got["service.beta.kubernetes.io/do-loadbalancer-tls-passthrough"] != "true" {
		t.Errorf("missing DO TLS passthrough annotation: %+v", got)
	}
}

func TestCloud_privateLoadBalancerAnnotations_isEmpty(t *testing.T) {
	t.Parallel()

	if got := newCloud("vpc", "proj").PrivateLoadBalancerAnnotations(); len(got) != 0 {
		t.Errorf("expected no private annotations, got %+v", got)
	}
}

func TestCloud_forDatabase_propagatesIDs(t *testing.T) {
	t.Parallel()

	d := newCloud("vpc-123", "proj-456").ForDatabase()

	assertStringInput(t, "VPCID", d.VPCID(), "vpc-123")
	assertStringInput(t, "ProjectID", d.ProjectID(), "proj-456")
}

func TestCloud_forObjectStorage_propagatesIDs(t *testing.T) {
	t.Parallel()

	o := newCloud("vpc-789", "proj-321").ForObjectStorage()

	assertStringInput(t, "VPCID", o.VPCID(), "vpc-789")
	assertStringInput(t, "ProjectID", o.ProjectID(), "proj-321")
}

// Each call must return a fresh handle so two callers don't end up
// sharing pointer-equal state. Pointer-identity is the strongest check
// available because the IDs are interface-typed and not comparable.
func TestCloud_forDatabase_freshPerCall(t *testing.T) {
	t.Parallel()

	c := newCloud("vpc", "proj")
	if c.ForDatabase() == c.ForDatabase() {
		t.Errorf("expected independent DatabaseCloud values, got the same pointer")
	}
}

func TestCloud_forObjectStorage_freshPerCall(t *testing.T) {
	t.Parallel()

	c := newCloud("vpc", "proj")
	if c.ForObjectStorage() == c.ForObjectStorage() {
		t.Errorf("expected independent ObjectStorageCloud values, got the same pointer")
	}
}

func assertStringInput(t *testing.T, name string, got pulumi.StringInput, want string) {
	t.Helper()

	s, ok := got.(pulumi.String)
	if !ok {
		t.Fatalf("%s: expected pulumi.String, got %T", name, got)
	}
	if string(s) != want {
		t.Errorf("%s: got %q, want %q", name, s, want)
	}
}
