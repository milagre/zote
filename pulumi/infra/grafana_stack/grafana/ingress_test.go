package grafana

import (
	"net/url"
	"testing"

	"github.com/milagre/zote/pulumi/infra"
)

func TestIngressHosts(t *testing.T) {
	t.Parallel()

	got := ingressHosts("grafana", "infra", []string{"wealthmode.test", "wealthmode.com"})
	want := []string{
		"grafana.infra.wealthmode.test",
		"grafana.infra.wealthmode.com",
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPublicURL(t *testing.T) {
	t.Parallel()

	got := publicURL("grafana.infra.wealthmode.com")
	want := url.URL{
		Scheme: "https",
		Host:   "grafana.infra.wealthmode.com",
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
