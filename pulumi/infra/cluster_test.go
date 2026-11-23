package infra

import "testing"

func TestCluster_setters_nilReceiver(t *testing.T) {
	t.Parallel()

	var c *Cluster
	c.SetTunnelIngressClass("cloudflare-tunnel")
	c.SetClusterIssuer("letsencrypt-http01")
	c.SetGrafana(nil)
}

func TestCluster_setters(t *testing.T) {
	t.Parallel()

	c := &Cluster{}
	c.SetTunnelIngressClass("cloudflare-tunnel")
	c.SetClusterIssuer("letsencrypt-http01")
	c.HasKeda = true

	if c.TunnelIngressClassName == nil || *c.TunnelIngressClassName != "cloudflare-tunnel" {
		t.Fatalf("TunnelIngressClassName = %v", c.TunnelIngressClassName)
	}

	if c.ClusterIssuerName == nil || *c.ClusterIssuerName != "letsencrypt-http01" {
		t.Fatalf("ClusterIssuerName = %v", c.ClusterIssuerName)
	}

	if !c.HasKeda {
		t.Fatal("HasKeda not set")
	}
}

func TestCluster_SetGateway_nilReceiver(t *testing.T) {
	t.Parallel()

	var c *Cluster
	c.SetGateway("gateway", "infra")
}

func TestGateway_AddHostnames_keepsFirstOfEach(t *testing.T) {
	t.Parallel()

	c := &Cluster{}
	c.SetGateway("gateway", "infra")
	c.Gateway.AddHostnames("a.example.com", "b.example.com")
	c.Gateway.AddHostnames("b.example.com", "c.example.com", "a.example.com")

	got := c.Gateway.Hostnames()
	want := []string{"a.example.com", "b.example.com", "c.example.com"}
	if len(got) != len(want) {
		t.Fatalf("Hostnames = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Hostnames = %v, want %v", got, want)
		}
	}
}
