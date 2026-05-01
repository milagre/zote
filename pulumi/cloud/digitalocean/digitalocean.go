// Package digitalocean implements [cloud.Cloud] (LB Service annotations) and per-DB handles via [Cloud.ForDatabase].
package digitalocean

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Cloud struct{}

func New() *Cloud {
	return &Cloud{}
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

// ForDatabase returns a per-DB handle; vpcID and projectID are [pulumi.StringInput] so stack outputs are allowed.
func (c *Cloud) ForDatabase(vpcID, projectID pulumi.StringInput) *DatabaseCloud {
	return &DatabaseCloud{
		parent:    c,
		vpcID:     vpcID,
		projectID: projectID,
	}
}

// DatabaseCloud implements database/digitalocean.Cloud for one VPC/project pair.
type DatabaseCloud struct {
	parent    *Cloud
	vpcID     pulumi.StringInput
	projectID pulumi.StringInput
}

func (d *DatabaseCloud) VPCID() pulumi.StringInput { return d.vpcID }

func (d *DatabaseCloud) ProjectID() pulumi.StringInput { return d.projectID }
