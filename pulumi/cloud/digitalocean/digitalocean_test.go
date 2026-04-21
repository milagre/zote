package digitalocean_test

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/cloud"
	"github.com/milagre/zote/pulumi/cloud/digitalocean"
	dbdo "github.com/milagre/zote/pulumi/database/digitalocean"
)

// compile-time check: digitalocean.Cloud satisfies cloud.Cloud.
var _ cloud.Cloud = (*digitalocean.Cloud)(nil)

// compile-time check: the value returned by ForDatabase satisfies the
// per-instance database interface.
var _ dbdo.Cloud = (*digitalocean.DatabaseCloud)(nil)

func TestCloud_publicLoadBalancerAnnotations(t *testing.T) {
	t.Parallel()

	got := digitalocean.New().PublicLoadBalancerAnnotations()

	if got["service.beta.kubernetes.io/do-loadbalancer-tls-ports"] != "443" {
		t.Errorf("missing DO TLS ports annotation: %+v", got)
	}
	if got["service.beta.kubernetes.io/do-loadbalancer-tls-passthrough"] != "true" {
		t.Errorf("missing DO TLS passthrough annotation: %+v", got)
	}
}

func TestCloud_privateLoadBalancerAnnotations_isEmpty(t *testing.T) {
	t.Parallel()

	if got := digitalocean.New().PrivateLoadBalancerAnnotations(); len(got) != 0 {
		t.Errorf("expected no private annotations, got %+v", got)
	}
}

func TestCloud_forDatabase_returnsConfiguredIDs(t *testing.T) {
	t.Parallel()

	// Raw pulumi.String literals are the only shape of
	// pulumi.StringInput whose concrete value can be inspected
	// synchronously; StringOutput values must be resolved via Apply at
	// runtime. Passing literals exercises the happy-path identity of
	// the stored input without spinning up a Pulumi program.
	d := digitalocean.New().ForDatabase(pulumi.String("vpc-123"), pulumi.String("proj-456"))

	vpc, ok := d.VPCID().(pulumi.String)
	if !ok {
		t.Fatalf("VPCID: expected pulumi.String, got %T", d.VPCID())
	}
	if string(vpc) != "vpc-123" {
		t.Errorf("VPCID: got %q, want %q", vpc, "vpc-123")
	}

	proj, ok := d.ProjectID().(pulumi.String)
	if !ok {
		t.Fatalf("ProjectID: expected pulumi.String, got %T", d.ProjectID())
	}
	if string(proj) != "proj-456" {
		t.Errorf("ProjectID: got %q, want %q", proj, "proj-456")
	}
}

func TestCloud_forDatabase_producesIndependentHandlesPerInstance(t *testing.T) {
	t.Parallel()

	c := digitalocean.New()
	a := c.ForDatabase(pulumi.String("vpc-a"), pulumi.String("proj-a"))
	b := c.ForDatabase(pulumi.String("vpc-b"), pulumi.String("proj-b"))

	// Each ForDatabase call must return a fresh DatabaseCloud; a
	// single shared handle would conflate two databases living in
	// different networks when the caller only wanted one per
	// instance. Pointer-identity is the strongest guarantee available
	// here because the IDs are interface-typed and not comparable via
	// `==` in the general case.
	if a == b {
		t.Errorf("expected independent DatabaseCloud values, got the same pointer")
	}

	aVPC, ok := a.VPCID().(pulumi.String)
	if !ok {
		t.Fatalf("a.VPCID: expected pulumi.String, got %T", a.VPCID())
	}
	bVPC, ok := b.VPCID().(pulumi.String)
	if !ok {
		t.Fatalf("b.VPCID: expected pulumi.String, got %T", b.VPCID())
	}
	if aVPC == bVPC {
		t.Errorf("expected independent VPCIDs, both were %q", aVPC)
	}

	aProj, ok := a.ProjectID().(pulumi.String)
	if !ok {
		t.Fatalf("a.ProjectID: expected pulumi.String, got %T", a.ProjectID())
	}
	bProj, ok := b.ProjectID().(pulumi.String)
	if !ok {
		t.Fatalf("b.ProjectID: expected pulumi.String, got %T", b.ProjectID())
	}
	if aProj == bProj {
		t.Errorf("expected independent ProjectIDs, both were %q", aProj)
	}
}
