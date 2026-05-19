package http

import (
	"testing"

	"github.com/milagre/zote/pulumi/infra"
)

func TestPrivateIngressClassName(t *testing.T) {
	t.Parallel()

	if got := privateIngressClassName(nil); got != nil {
		t.Fatalf("nil cluster: got %v", got)
	}

	c := &infra.Cluster{}
	c.SetPublicIngressClass("nginx")
	got := privateIngressClassName(c)
	if got == nil || *got != "nginx" {
		t.Fatalf("public fallback: got %v", got)
	}

	c.SetPrivateIngressClass("nginx-private")
	got = privateIngressClassName(c)
	if got == nil || *got != "nginx-private" {
		t.Fatalf("private: got %v", got)
	}
}

func TestPublicIngressClassName(t *testing.T) {
	t.Parallel()

	c := &infra.Cluster{}
	if publicIngressClassName(c) != nil {
		t.Fatal("expected nil before registration")
	}

	c.SetPublicIngressClass("nginx")
	if got := publicIngressClassName(c); got == nil || *got != "nginx" {
		t.Fatalf("got %v", got)
	}
}

func TestClusterIssuerName(t *testing.T) {
	t.Parallel()

	c := &infra.Cluster{}
	c.SetClusterIssuer("letsencrypt-http01")
	if got := clusterIssuerName(c); got == nil || *got != "letsencrypt-http01" {
		t.Fatalf("got %v", got)
	}
}

func TestTunnelIngressClassName(t *testing.T) {
	t.Parallel()

	c := &infra.Cluster{}
	c.SetTunnelIngressClass("cloudflare-tunnel")
	if got := tunnelIngressClassName(c); got == nil || *got != "cloudflare-tunnel" {
		t.Fatalf("got %v", got)
	}
}
