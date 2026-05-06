// Package annotations holds Pulumi Kubernetes metadata annotation keys and values.
package annotations

const (
	PatchForceKey = "pulumi.com/patchForce"

	SkipAwaitKey = "pulumi.com/skipAwait"

	// SkipAwaitValueAll is the literal "true"; required for builtin Deployment rollout
	// and Job completion awaits to short-circuit. Also skips delete-await for kinds in
	// allowsSkipAwaitWithDelete (Deployments, Jobs, …).
	SkipAwaitValueAll = "true"

	// WaitForKey overrides the builtin per-kind awaiter with a user-supplied
	// condition. Setting any value short-circuits the kind's default await
	// (e.g. Job-success, Deployment rollout) without touching delete-await,
	// which keeps DeleteBeforeReplace ordering correct.
	WaitForKey = "pulumi.com/waitFor"

	// WaitForValueImmediate is a JSONPath that the apiserver fills in
	// synchronously on POST, so the await resolves on first poll.
	WaitForValueImmediate = "jsonpath={.metadata.uid}"
)
