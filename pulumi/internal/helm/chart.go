// Package helm installs charts via helm/v3 Release (real Helm ordering/hooks), not helm/v4 Chart (per-manifest SSA).
package helm

import (
	"fmt"
	"strings"

	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/tokens"
)

type ChartSpec struct {
	TypeToken      string
	Chart          string
	Repository     string // empty when Chart is a full OCI ref
	DefaultVersion string
}

type ChartArgs struct {
	Namespace string
	Version   *string // nil → ChartSpec.DefaultVersion
	Values    pulumi.Map
}

// OptionalChartVersion yields nil when version is whitespace-only so RegisterChart picks [ChartSpec.DefaultVersion].
func OptionalChartVersion(version string) *string {
	v := strings.TrimSpace(version)
	if v == "" {
		return nil
	}

	return &v
}

// ChartComponent embeds the installed [helmv3.Release] for DependsOn from sibling resources.
type ChartComponent struct {
	pulumi.ResourceState

	Release *helmv3.Release
}

// RegisterChartComponentNamed registers comp as a Pulumi component only (no Helm release),
// using logicalName as the resource name segment (see [tokens.Qualify] for the usual pattern).
func RegisterChartComponentNamed(
	ctx *pulumi.Context,
	logicalName string,
	spec ChartSpec,
	comp *ChartComponent,
	opts ...pulumi.ResourceOption,
) error {
	if ctx == nil {
		return fmt.Errorf("%s: pulumi context is required", spec.TypeToken)
	}
	if logicalName == "" {
		return fmt.Errorf("%s: component logical name is required", spec.TypeToken)
	}
	if comp == nil {
		return fmt.Errorf("%s: comp is required", spec.TypeToken)
	}

	if err := ctx.RegisterComponentResource(spec.TypeToken, logicalName, comp, opts...); err != nil {
		return fmt.Errorf("registering %s: %w", spec.TypeToken, err)
	}

	return nil
}

// RegisterChartComponent registers comp as a Pulumi component only (no Helm release).
// Call this before creating other resources parented to comp (e.g. secrets) so the parent chain is valid.
// The component logical name is [tokens.Qualify](namespace, name).
func RegisterChartComponent(
	ctx *pulumi.Context,
	namespace, name string,
	spec ChartSpec,
	comp *ChartComponent,
	opts ...pulumi.ResourceOption,
) error {
	if namespace == "" {
		return fmt.Errorf("%s: Namespace is required", spec.TypeToken)
	}

	return RegisterChartComponentNamed(ctx, tokens.Qualify(namespace, name), spec, comp, opts...)
}

// InstallChart installs spec.Chart under comp. comp must already be registered (e.g. via [RegisterChartComponent]).
func InstallChart(
	ctx *pulumi.Context,
	namespace, name string,
	spec ChartSpec,
	args *ChartArgs,
	comp *ChartComponent,
) error {
	if ctx == nil {
		return fmt.Errorf("%s: pulumi context is required", spec.TypeToken)
	}
	if args == nil {
		return fmt.Errorf("%s: args is required", spec.TypeToken)
	}
	if comp == nil {
		return fmt.Errorf("%s: comp is required", spec.TypeToken)
	}
	if args.Namespace == "" {
		return fmt.Errorf("%s: Namespace is required", spec.TypeToken)
	}
	if args.Namespace != namespace {
		return fmt.Errorf("%s: args.Namespace must match install namespace", spec.TypeToken)
	}

	resourceName := tokens.Qualify(namespace, name)

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

// RegisterChart registers comp, then installs spec.Chart; name is both [tokens.Qualify] prefix and Helm release name.
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
	if err := RegisterChartComponent(ctx, args.Namespace, name, spec, comp, opts...); err != nil {
		return err
	}

	return InstallChart(ctx, args.Namespace, name, spec, args, comp)
}
