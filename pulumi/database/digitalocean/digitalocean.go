// Package digitalocean defines the interface that DigitalOcean-backed
// database components consume to learn where a particular database
// instance should be placed.
//
// The interface is per-instance rather than per-provider: two databases
// in the same Pulumi program can belong to different VPCs or projects,
// so the VPC/project identifiers cannot live on a shared cloud-provider
// singleton. A caller obtains a value satisfying this interface by
// calling cloud/digitalocean.Cloud.ForDatabase(vpcID, projectID), which
// pairs the per-instance network context with the shared provider
// object and returns the combined handle.
package digitalocean

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Cloud is the handle a DigitalOcean-backed database backend uses to
// resolve network placement for a single managed database cluster.
// Each instance receives its own Cloud; the values are not
// interchangeable between instances that differ in VPC or project.
//
// The IDs are returned as pulumi.StringInput rather than plain Go
// strings because callers that construct cloud resources in the same
// program only hold their IDs as pulumi.Output values — an Input-typed
// API accepts both that shape and raw string literals
// (pulumi.String("…")), so consumers wire the same interface for
// zero-state construction and for ID-known configurations.
type Cloud interface {
	// VPCID is the UUID of the VPC the database cluster joins.
	VPCID() pulumi.StringInput

	// ProjectID is the UUID of the DigitalOcean project the cluster is
	// billed under.
	ProjectID() pulumi.StringInput
}
