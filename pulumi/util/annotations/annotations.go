// Package annotations holds Pulumi Kubernetes metadata annotation keys and values.
package annotations

const (
	PatchForceKey = "pulumi.com/patchForce"

	SkipAwaitKey      = "pulumi.com/skipAwait"
	SkipAwaitValueAll = "true"

	WaitForKey            = "pulumi.com/waitFor"
	WaitForValueImmediate = "jsonpath={.metadata.uid}"
)
