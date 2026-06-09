package infra

import (
	"github.com/pulumiverse/pulumi-grafana/sdk/go/grafana"
)

// Cluster holds deployed cluster-scoped capabilities for autodiscovery by other
// components (ingress classes, KEDA, Grafana provider, …). It must not import
// sibling infra/* packages (import cycles).
type Cluster struct {
	HasKeda bool

	// Grafana is the Pulumi Grafana provider aimed at the in-cluster dashboard API.
	Grafana *grafana.Provider

	// PublicIngressClassName is the ingress class for public traffic.
	PublicIngressClassName *string

	// PublicIngressServiceName is the controller Service name in the ingress namespace.
	PublicIngressServiceName *string

	// PublicIngressServiceHostname is the in-cluster hostname of PublicIngressServiceName.
	PublicIngressServiceHostname *string

	// PrivateIngressClassName is reserved for a future private ingress controller.
	PrivateIngressClassName *string

	// TunnelIngressClassName is the cloudflare-tunnel IngressClass when installed.
	TunnelIngressClassName *string

	// ClusterIssuerName is the cert-manager ClusterIssuer for TLS Ingresses.
	ClusterIssuerName *string
}

func stringPtr(s string) *string {
	return &s
}

// SetPublicIngressClass records the public ingress class name.
func (c *Cluster) SetPublicIngressClass(name string) {
	if c == nil {
		return
	}

	c.PublicIngressClassName = stringPtr(name)
}

// SetPublicIngressService records the public ingress controller Service name and hostname.
func (c *Cluster) SetPublicIngressService(serviceName, namespace string) {
	if c == nil {
		return
	}

	c.PublicIngressServiceName = stringPtr(serviceName)
	c.PublicIngressServiceHostname = stringPtr(serviceName + "." + namespace + ".svc.cluster.local")
}

// SetPrivateIngressClass records the private ingress class name.
func (c *Cluster) SetPrivateIngressClass(name string) {
	if c == nil {
		return
	}

	c.PrivateIngressClassName = stringPtr(name)
}

// SetTunnelIngressClass records the tunnel ingress class name.
func (c *Cluster) SetTunnelIngressClass(name string) {
	if c == nil {
		return
	}

	c.TunnelIngressClassName = stringPtr(name)
}

// SetClusterIssuer records the cert-manager ClusterIssuer name.
func (c *Cluster) SetClusterIssuer(name string) {
	if c == nil {
		return
	}

	c.ClusterIssuerName = stringPtr(name)
}

// SetGrafana records the Pulumi Grafana provider for the deployed dashboard.
func (c *Cluster) SetGrafana(p *grafana.Provider) {
	if c == nil {
		return
	}

	c.Grafana = p
}
