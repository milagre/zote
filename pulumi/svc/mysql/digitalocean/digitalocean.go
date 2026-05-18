// Package digitalocean is per-DB placement (VPC + project); use cloud/digitalocean.Cloud.ForDatabase.
package digitalocean

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Cloud interface {
	VPCID() pulumi.StringInput
	ProjectID() pulumi.StringInput
}
