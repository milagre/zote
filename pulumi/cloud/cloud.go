// Package cloud describes the cross-cutting contract every concrete cloud
// provider package (cloud/digitalocean, and future cloud/aws etc.) is
// expected to satisfy. The consumer Pulumi program selects one provider
// for its stack and passes it down to cloud-polymorphic components.
package cloud

// Cloud is the umbrella interface every cloud provider implementation
// satisfies. It intentionally only contains provider-wide capabilities
// that are not tied to any single resource family. Resource-family
// capabilities (MySQL, Redis, object storage, etc.) are expressed as
// separate interfaces declared alongside the components that consume
// them, and a provider type like digitalocean.Cloud satisfies each one
// implicitly as those features are added.
type Cloud interface {
	// PublicLoadBalancerAnnotations returns the Kubernetes Service
	// annotations required to expose a LoadBalancer service to the
	// public internet on this cloud.
	PublicLoadBalancerAnnotations() map[string]string

	// PrivateLoadBalancerAnnotations returns the Kubernetes Service
	// annotations required to expose a LoadBalancer service to the
	// cloud-internal network only.
	PrivateLoadBalancerAnnotations() map[string]string
}
