// Package annotations is Pulumi k8s metadata helpers ([Managed], [PatchForce]).
package annotations

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

const (
	SkipAwaitKey  = "pulumi.com/skipAwait"
	PatchForceKey = "pulumi.com/patchForce"
)

func Managed() pulumi.StringMap {
	return pulumi.StringMap{
		// SkipAwaitReady skips readiness wait on create/update only (Kubernetes
		// provider v4.18+). Deletion is still awaited.
		SkipAwaitKey: pulumi.String("ready"),
	}
}

// ManagedWith returns [Managed] annotations overlaid with extra. Keys in
// extra replace the same key from Managed.
func ManagedWith(extra pulumi.StringMap) pulumi.StringMap {
	base := Managed()
	if len(extra) == 0 {
		return base
	}
	out := make(pulumi.StringMap, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func PatchForce() pulumi.StringMap {
	return pulumi.StringMap{
		PatchForceKey: pulumi.String("true"),
	}
}
