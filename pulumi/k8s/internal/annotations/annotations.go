// Package annotations holds Pulumi Kubernetes metadata annotation keys and values.
package annotations

const (
	PatchForceKey = "pulumi.com/patchForce"

	SkipAwaitKey = "pulumi.com/skipAwait"

	// SkipAwaitValueReady skips readiness wait on create/update only (Kubernetes
	// provider v4.18+). Deletion is still awaited.
	SkipAwaitValueReady = "ready"

	// SkipAwaitValueAll skips provider await on create, update, and delete.
	SkipAwaitValueAll = "true"
)
