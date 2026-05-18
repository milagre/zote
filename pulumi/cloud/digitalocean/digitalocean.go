// Package digitalocean is the DO cloud-provider handle: LB Service annotations and per-resource scopes (database, object storage).
package digitalocean

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Cloud carries the VPC + project IDs every cloud-scoped DO resource
// needs. ForDatabase / ForObjectStorage build per-resource handles
// from these IDs without re-asking the caller.
type Cloud struct {
	vpcID     pulumi.StringInput
	projectID pulumi.StringInput
}

// New takes [pulumi.StringInput] so VPC/project stack outputs are
// allowed; the IDs are deferred to apply-scoped data-source invokes.
func New(vpcID, projectID pulumi.StringInput) *Cloud {
	return &Cloud{vpcID: vpcID, projectID: projectID}
}

func (*Cloud) PublicLoadBalancerAnnotations() map[string]string {
	return map[string]string{
		"service.beta.kubernetes.io/do-loadbalancer-tls-ports":       "443",
		"service.beta.kubernetes.io/do-loadbalancer-tls-passthrough": "true",
	}
}

func (*Cloud) PrivateLoadBalancerAnnotations() map[string]string {
	return map[string]string{}
}

func (c *Cloud) ForDatabase() *DatabaseCloud {
	return &DatabaseCloud{vpcID: c.vpcID, projectID: c.projectID}
}

func (c *Cloud) ForObjectStorage() *ObjectStorageCloud {
	return &ObjectStorageCloud{vpcID: c.vpcID, projectID: c.projectID}
}

// DatabaseCloud satisfies svc/mysql/digitalocean.Cloud.
type DatabaseCloud struct {
	vpcID     pulumi.StringInput
	projectID pulumi.StringInput
}

func (d *DatabaseCloud) VPCID() pulumi.StringInput { return d.vpcID }

func (d *DatabaseCloud) ProjectID() pulumi.StringInput { return d.projectID }

// ObjectStorageCloud satisfies svc/objectstorage/digitalocean.Cloud.
type ObjectStorageCloud struct {
	vpcID     pulumi.StringInput
	projectID pulumi.StringInput
}

func (o *ObjectStorageCloud) VPCID() pulumi.StringInput { return o.vpcID }

func (o *ObjectStorageCloud) ProjectID() pulumi.StringInput { return o.projectID }
