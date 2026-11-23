package infra

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-grafana/sdk/go/grafana"
)

// GatewayHTTPListener names the shared Gateway's plain HTTP listener.
const GatewayHTTPListener = "http"

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

	// MetricsQueryURL is the Prometheus-compatible query endpoint of the deployed metrics
	// store, used by KEDA prometheus-trigger autoscalers.
	// Nil until the stats stack registers it.
	MetricsQueryURL *string

	// PrivateIngressClassName is reserved for a future private ingress controller.
	PrivateIngressClassName *string

	// TunnelIngressClassName is the cloudflare-tunnel IngressClass when installed.
	TunnelIngressClassName *string

	// ClusterIssuerName is the cert-manager ClusterIssuer for TLS Ingresses.
	ClusterIssuerName *string

	// Gateway is the cluster's shared entry point for routed traffic.
	Gateway *Gateway
}

// Gateway identifies the cluster's shared entry point for routed traffic.
// Routes from any namespace can attach to it. It has only a plain HTTP
// listener; where TLS is terminated, each workload attaches its HTTPS
// listeners in a ListenerSet of its own.
type Gateway struct {
	Name      string
	Namespace string

	// Ready holds the resources a route must wait on before it is created.
	Ready []pulumi.Resource
}

// GatewayListenerName is the name of host's HTTPS listener.
func GatewayListenerName(host string) string {
	return host
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
