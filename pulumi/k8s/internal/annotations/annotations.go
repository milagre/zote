// Package annotations is Pulumi k8s metadata helpers ([Managed], [PatchForce]).
package annotations

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

const (
	SkipAwaitKey  = "pulumi.com/skipAwait"
	PatchForceKey = "pulumi.com/patchForce"
)

func Managed() pulumi.StringMap {
	return pulumi.StringMap{
		SkipAwaitKey: pulumi.String("true"),
	}
}

func PatchForce() pulumi.StringMap {
	return pulumi.StringMap{
		PatchForceKey: pulumi.String("true"),
	}
}
