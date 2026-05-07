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
