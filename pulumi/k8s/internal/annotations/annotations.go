// Package annotations holds well-known Pulumi Kubernetes metadata keys
// and helpers for the annotations zote applies to managed API objects.
// Callers may compose PatchForce() with their own maps when they choose
// to opt into SSA force-patch behavior; zote does not add it by default.
package annotations

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

const (
	// SkipAwaitKey opts a resource out of provider-side readiness waits
	// (rollout, job completion, etc.).
	SkipAwaitKey = "pulumi.com/skipAwait"

	// PatchForceKey is the Pulumi Kubernetes server-side apply annotation
	// that forces patch application when field managers conflict (often
	// described as “force patch” in provider docs).
	PatchForceKey = "pulumi.com/patchForce"
)

// Managed returns the default Pulumi annotations zote applies to its k8s
// workloads so updates do not block on readiness.
func Managed() pulumi.StringMap {
	return pulumi.StringMap{
		SkipAwaitKey: pulumi.String("true"),
	}
}

// PatchForce returns only the patch-force annotation map, for callers
// that want to merge it into resource metadata alongside other keys.
func PatchForce() pulumi.StringMap {
	return pulumi.StringMap{
		PatchForceKey: pulumi.String("true"),
	}
}
