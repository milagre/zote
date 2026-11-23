package infra

import (
	"slices"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-grafana/sdk/go/grafana"
)

// GatewayHTTPListener names the shared Gateway's plain HTTP listener.
const GatewayHTTPListener = "http"

// Cluster holds deployed cluster-scoped capabilities for autodiscovery by other
// components (the Gateway, KEDA, Grafana provider, …). It must not import
// sibling infra/* packages (import cycles).
type Cluster struct {
	HasKeda bool

	// Grafana is the Pulumi Grafana provider aimed at the in-cluster dashboard API.
	Grafana *grafana.Provider

	// MetricsQueryURL is the Prometheus-compatible query endpoint of the deployed metrics
	// store, used by KEDA prometheus-trigger autoscalers.
	// Nil until the stats stack registers it.
	MetricsQueryURL *string

	// TunnelIngressClassName is the cloudflare-tunnel IngressClass when installed.
	TunnelIngressClassName *string

	// ClusterIssuerName is the cert-manager ClusterIssuer for the Gateway's
	// certificates.
	ClusterIssuerName *string

	// Gateway is the cluster's shared entry point for routed traffic.
	Gateway *Gateway
}

// Gateway identifies the cluster's shared entry point for routed traffic.
// Routes from any namespace can attach to it. Where TLS is terminated, each
// hostname gets an HTTPS listener of its own, so the listeners come from the
// hostnames routes register, and the Gateway is created once every route has
// registered.
type Gateway struct {
	Name      string
	Namespace string

	// Ready holds the resources a route must wait on before it is created.
	Ready []pulumi.Resource

	hostnames []string
}

// GatewayListenerName is the name of host's HTTPS listener.
func GatewayListenerName(host string) string {
	return host
}

// AddHostnames records hosts as needing a listener. A host already recorded
// keeps its place.
func (g *Gateway) AddHostnames(hosts ...string) {
	for _, h := range hosts {
		if !slices.Contains(g.hostnames, h) {
			g.hostnames = append(g.hostnames, h)
		}
	}
}

// Hostnames returns the recorded hostnames in the order they were first added.
func (g *Gateway) Hostnames() []string {
	return slices.Clone(g.hostnames)
}

func stringPtr(s string) *string {
	return &s
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

// SetGateway records the shared Gateway; routes created before ready finishes
// fail, so they wait on it.
func (c *Cluster) SetGateway(name, namespace string, ready ...pulumi.Resource) {
	if c == nil {
		return
	}

	c.Gateway = &Gateway{
		Name:      name,
		Namespace: namespace,
		Ready:     ready,
	}
}

// SetGrafana records the Pulumi Grafana provider for the deployed dashboard.
func (c *Cluster) SetGrafana(p *grafana.Provider) {
	if c == nil {
		return
	}

	c.Grafana = p
}

// SetMetricsQueryURL records the Prometheus-compatible query endpoint of the
// deployed metrics store.
func (c *Cluster) SetMetricsQueryURL(url string) {
	if c == nil {
		return
	}

	c.MetricsQueryURL = stringPtr(url)
}
