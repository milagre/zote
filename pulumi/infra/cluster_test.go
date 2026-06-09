package infra

import "testing"

func TestCluster_setters_nilReceiver(t *testing.T) {
	t.Parallel()

	var c *Cluster
	c.SetPublicIngressClass("nginx")
	c.SetPublicIngressService("ingress-nginx-controller", "infra")
	c.SetTunnelIngressClass("cloudflare-tunnel")
	c.SetClusterIssuer("letsencrypt-http01")
	c.SetGrafana(nil)
}

func TestCluster_setters(t *testing.T) {
	t.Parallel()

	c := &Cluster{}
	c.SetPublicIngressClass("nginx")
	c.SetPublicIngressService("ingress-nginx-controller", "infra")
	c.SetTunnelIngressClass("cloudflare-tunnel")
	c.SetClusterIssuer("letsencrypt-http01")
	c.HasKeda = true

	if c.PublicIngressClassName == nil || *c.PublicIngressClassName != "nginx" {
		t.Fatalf("PublicIngressClassName = %v, want nginx", c.PublicIngressClassName)
	}

	if c.PublicIngressServiceName == nil || *c.PublicIngressServiceName != "ingress-nginx-controller" {
		t.Fatalf("PublicIngressServiceName = %v, want ingress-nginx-controller", c.PublicIngressServiceName)
	}

	wantHostname := "ingress-nginx-controller.infra.svc.cluster.local"
	if c.PublicIngressServiceHostname == nil || *c.PublicIngressServiceHostname != wantHostname {
		t.Fatalf("PublicIngressServiceHostname = %v, want %s", c.PublicIngressServiceHostname, wantHostname)
	}

	if c.PrivateIngressClassName != nil {
		t.Fatal("PrivateIngressClassName should stay nil until a private controller exists")
	}

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
