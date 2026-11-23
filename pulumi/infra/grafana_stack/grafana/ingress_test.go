package grafana

import (
	"net/url"
	"testing"

	"github.com/milagre/zote/pulumi/infra"
)

func TestPublicURL(t *testing.T) {
	t.Parallel()

	got := publicURL("grafana.infra.example.com")
	want := url.URL{
		Scheme: "https",
		Host:   "grafana.infra.example.com",
		Path:   "/",
	}
	if got != want {
		t.Fatalf("publicURL = %#v, want %#v", got, want)
	}
}

func TestPublicIngressClassName(t *testing.T) {
	t.Parallel()

	if publicIngressClassName(nil) != nil {
		t.Fatal("nil cluster: expected nil")
	}

	c := &infra.Cluster{}
	if publicIngressClassName(c) != nil {
		t.Fatal("expected nil before registration")
	}

	c.SetPublicIngressClass("nginx")
	if got := publicIngressClassName(c); got == nil || *got != "nginx" {
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

func TestClusterIssuerName(t *testing.T) {
	t.Parallel()

	c := &infra.Cluster{}
	c.SetClusterIssuer("letsencrypt-http01")
	if got := clusterIssuerName(c); got == nil || *got != "letsencrypt-http01" {
		t.Fatalf("got %v", got)
	}
}
