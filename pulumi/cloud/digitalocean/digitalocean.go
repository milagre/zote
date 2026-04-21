// Package digitalocean is the DigitalOcean implementation of the
// library's cross-cutting cloud provider contract (see package cloud).
//
// The value is deliberately narrow: it carries only the capabilities
// every DigitalOcean stack needs regardless of which resources the
// caller ends up creating (currently the load-balancer annotations).
// Per-resource context that varies from one instance to the next is
// produced by methods on this type that return per-instance handles
// (for example ForDatabase), so a single Cloud value can serve multiple
// databases, clusters, or buckets that live in different VPCs or
// projects.
package digitalocean

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Cloud is the DigitalOcean provider handle shared across every
// resource a Pulumi program declares on DigitalOcean.
type Cloud struct{}

// New returns a DigitalOcean cloud provider handle.
func New() *Cloud {
	return &Cloud{}
}

// PublicLoadBalancerAnnotations returns the Kubernetes Service
// annotations required to expose a LoadBalancer service to the public
// internet on DigitalOcean.
func (*Cloud) PublicLoadBalancerAnnotations() map[string]string {
	return map[string]string{
		"service.beta.kubernetes.io/do-loadbalancer-tls-ports":       "443",
		"service.beta.kubernetes.io/do-loadbalancer-tls-passthrough": "true",
	}
}

// PrivateLoadBalancerAnnotations returns the Kubernetes Service
// annotations required to expose a LoadBalancer service only to the
// VPC. DigitalOcean does not currently require additional annotations
// for that case, but the method is part of the shared contract.
func (*Cloud) PrivateLoadBalancerAnnotations() map[string]string {
	return map[string]string{}
}

// ForDatabase returns a per-instance handle for placing a single
// DigitalOcean-managed database cluster in vpcID and billing it to
// projectID. Callers obtain one DatabaseCloud per database instance
// rather than reusing a single value; two databases in different VPCs
// receive two different DatabaseCloud values from the same Cloud.
//
// vpcID and projectID are typed as pulumi.StringInput (rather than
// plain Go strings) because in a zero-state Pulumi program these IDs
// are outputs of resources created earlier in the same program — they
// cannot be known synchronously at component-construction time. A
// pulumi.StringInput accepts both raw string literals (pulumi.String)
// and Output values (pulumi.StringOutput), so callers with either
// shape of ID can supply it here; the DigitalOcean database backend
// defers ID-dependent work into an Apply-scoped data-source invocation
// so it works regardless of which shape arrives.
func (c *Cloud) ForDatabase(vpcID, projectID pulumi.StringInput) *DatabaseCloud {
	return &DatabaseCloud{
		parent:    c,
		vpcID:     vpcID,
		projectID: projectID,
	}
}

// DatabaseCloud is the per-instance handle returned by
// Cloud.ForDatabase. It satisfies database/digitalocean.Cloud
// implicitly so a database component can take the interface type and
// accept this value without an import cycle.
//
// The parent reference is kept so future capabilities that rely on
// provider-wide state (credentials, an explicit pulumi-digitalocean
// provider resource, account-level lookups) can be exposed as
// additional methods without changing the interface consumers already
// depend on.
type DatabaseCloud struct {
	parent    *Cloud
	vpcID     pulumi.StringInput
	projectID pulumi.StringInput
}

// VPCID returns the (possibly deferred) UUID of the VPC this database
// instance should join.
func (d *DatabaseCloud) VPCID() pulumi.StringInput { return d.vpcID }

// ProjectID returns the (possibly deferred) UUID of the DigitalOcean
// project this database instance is billed under.
func (d *DatabaseCloud) ProjectID() pulumi.StringInput { return d.projectID }
