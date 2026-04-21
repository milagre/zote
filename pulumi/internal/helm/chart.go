// Package helm holds internal helpers for Zote components that wrap
// Helm charts. Every wrapper installs its chart through the helm/v3
// Release resource rather than the helm/v4 Chart resource: the former
// drives Helm as a real install (hooks fire in the documented order,
// pre-install Jobs that patch admission webhook caBundles run before
// the consuming Deployment schedules, release state is tracked as a
// single Pulumi resource), while the latter is a manifest renderer
// that fans every rendered manifest out as an individual Pulumi
// resource and strips Helm's hook ordering. The hook-driven charts in
// this library (ingress-nginx, kube-prometheus-stack, grafana,
// cert-manager) need Helm's runtime to come up correctly on any
// cluster; the render-and-track-each-manifest model of helm/v4 simply
// does not apply them in a working order.
package helm

import (
	"fmt"

	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/tokens"
)

// ChartSpec describes a single Helm chart install that a zote infra
// component wraps.
type ChartSpec struct {
	// TypeToken is the Pulumi type token for the wrapping ComponentResource
	// (e.g. "zote:infra:Grafana"). Must be stable across releases.
	TypeToken string
	// Chart is the Helm chart name (e.g. "grafana").
	Chart string
	// Repository is the chart repository URL. May be empty for OCI-registry
	// charts where Chart already includes the full reference.
	Repository string
	// DefaultVersion is used when args.Version is nil.
	DefaultVersion string
}

// ChartArgs are the caller-supplied inputs common to Helm-chart-only
// components. Every field is a plain Go (or concrete pulumi) type
// rather than an Input interface: chart wiring in this library is
// always known synchronously — namespaces are compile-time constants,
// version pins are version strings from config, and Values are
// constructed inline. Inputs are not lost: the Values map still
// accepts pulumi.Input values for individual entries (secrets, ids),
// which is what one actually needs at runtime.
type ChartArgs struct {
	// Namespace is the target namespace. Required.
	Namespace string
	// Version overrides the component's default chart version. Nil
	// falls back to ChartSpec.DefaultVersion.
	Version *string
	// Values are the Helm chart values.
	Values pulumi.Map
}

// ChartComponent is the common fields every helm-chart wrapper embeds.
// Release is the installed helm/v3 Release resource; sibling code in
// this library sometimes needs it to express a pulumi.DependsOn on the
// whole install (e.g. cert-manager's ClusterIssuer depends on the
// cert-manager release before its CRDs are usable).
type ChartComponent struct {
	pulumi.ResourceState

	Release *helmv3.Release
}

// RegisterChart installs the chart as a child of the given component.
// The ComponentResource is registered inside this helper so every
// wrapper has identical lifecycle/output behavior. The chart is
// installed through the helm/v3 Release resource; the function keeps
// the name RegisterChart because at the call-site level every wrapper
// is still "install a chart", and the Release-vs-Chart distinction is
// an internal transport choice documented on the package.
//
// name is the bare logical name the caller picked for the component
// (e.g. "grafana", "cert-manager"). It flows in two directions:
//
//   - Pulumi URN: qualified to "<namespace>-<name>" via tokens.Qualify
//     so two namespaces that install the same chart get distinct URNs.
//     Used for both the wrapping ComponentResource and the helmv3.Release
//     child.
//   - Helm release name: passed through untouched. The release name ends
//     up in on-cluster object names (<release>-<chart>-…); renaming an
//     existing release is destructive (uninstall + reinstall), so the
//     on-cluster identity stays pinned to the bare name the caller
//     picked.
func RegisterChart(
	ctx *pulumi.Context,
	name string,
	spec ChartSpec,
	args *ChartArgs,
	comp *ChartComponent,
	opts ...pulumi.ResourceOption,
) error {
	if args == nil {
		return fmt.Errorf("%s: args is required", spec.TypeToken)
	}
	if args.Namespace == "" {
		return fmt.Errorf("%s: Namespace is required", spec.TypeToken)
	}

	resourceName := tokens.Qualify(args.Namespace, name)

	if err := ctx.RegisterComponentResource(spec.TypeToken, resourceName, comp, opts...); err != nil {
		return fmt.Errorf("registering %s: %w", spec.TypeToken, err)
	}

	var version pulumi.StringPtrInput
	switch {
	case args.Version != nil:
		version = pulumi.String(*args.Version).ToStringPtrOutput()
	case spec.DefaultVersion != "":
		version = pulumi.String(spec.DefaultVersion).ToStringPtrOutput()
	}

	releaseArgs := &helmv3.ReleaseArgs{
		Chart:     pulumi.String(spec.Chart),
		Name:      pulumi.String(name).ToStringPtrOutput(),
		Namespace: pulumi.String(args.Namespace).ToStringPtrOutput(),
		Version:   version,
		Values:    args.Values,
	}
	if spec.Repository != "" {
		releaseArgs.RepositoryOpts = &helmv3.RepositoryOptsArgs{
			Repo: pulumi.String(spec.Repository).ToStringPtrOutput(),
		}
	}

	rel, err := helmv3.NewRelease(ctx, resourceName, releaseArgs, pulumi.Parent(comp))
	if err != nil {
		return fmt.Errorf("creating %s release %q: %w", spec.TypeToken, spec.Chart, err)
	}

	comp.Release = rel

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{}); err != nil {
		return fmt.Errorf("registering outputs for %s: %w", spec.TypeToken, err)
	}

	return nil
}
