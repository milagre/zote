// Package digitalocean is per-bucket placement (VPC + project); use cloud/digitalocean.Cloud.ForObjectStorage.
package digitalocean

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Cloud interface {
	VPCID() pulumi.StringInput
	ProjectID() pulumi.StringInput
}
