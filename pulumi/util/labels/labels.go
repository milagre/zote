// Package labels provides standard Kubernetes labels for zote workloads.
package labels

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

const (
	KeyService = "service"
	KeyName    = "name"
)

// Pod returns service and name labels for StatefulSets, Deployments, and selectors.
func Pod(service, name string) pulumi.StringMap {
	return pulumi.StringMap{
		KeyService: pulumi.String(service),
		KeyName:    pulumi.String(name),
	}
}
