// Package loki deploys Grafana Loki; storage is local PV (SingleBinary) or managed object storage (SimpleScalable).
package loki

import (
	"fmt"

	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/cloud"
	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra/grafana/loki/internal/digitalocean"
	"github.com/milagre/zote/pulumi/infra/grafana/loki/internal/local"
	"github.com/milagre/zote/pulumi/profile"
	"github.com/milagre/zote/pulumi/tokens"
)

const (
	chart          = "loki"
	repository     = "https://grafana.github.io/helm-charts"
	defaultVersion = "7.0.0"
)

var typeToken = tokens.Token("infra", "Loki")

// Args bundles the YAML-decoded Config with the multi-provider Cloud
// container. Cloud is consulted only when Config.Cloud is populated;
// the provider field matching Config.Cloud must be non-nil.
type Args struct {
	Env       env.Env
	Namespace string

	Config Config
	Cloud  cloud.Cloud
}

type Loki struct {
	pulumi.ResourceState

	Release *helmv3.Release
}

// backend renders chart values and creates the side resources those
// values close over; returned resources flow into pulumi.DependsOn.
type backend interface {
	Setup(ctx *pulumi.Context, parent pulumi.Resource) (pulumi.Map, []pulumi.Resource, error)
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Loki, error) {
	if args == nil {
		return nil, fmt.Errorf("%s: args is required", typeToken)
	}
	if err := args.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	resourceName := tokens.Qualify(args.Namespace, name)

	comp := &Loki{}
	if err := ctx.RegisterComponentResource(typeToken, resourceName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	be, err := args.dispatch(name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	values, deps, err := be.Setup(ctx, comp)
	if err != nil {
		return nil, fmt.Errorf("%s: backend: %w", typeToken, err)
	}

	version := pulumi.String(defaultVersion).ToStringPtrOutput()
	if args.Config.Version != "" {
		version = pulumi.String(args.Config.Version).ToStringPtrOutput()
	}

	rel, err := helmv3.NewRelease(ctx, resourceName, &helmv3.ReleaseArgs{
		Chart:     pulumi.String(chart),
		Name:      pulumi.String(name).ToStringPtrOutput(),
		Namespace: pulumi.String(args.Namespace).ToStringPtrOutput(),
		Version:   version,
		RepositoryOpts: &helmv3.RepositoryOptsArgs{
			Repo: pulumi.String(repository).ToStringPtrOutput(),
		},
		Values: values,
	}, pulumi.Parent(comp), pulumi.DependsOn(deps))
	if err != nil {
		return nil, fmt.Errorf("%s: helm release: %w", typeToken, err)
	}

	comp.Release = rel

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

func (a *Args) dispatch(name string) (backend, error) {
	prof, err := profile.New(a.Config.Profile)
	if err != nil {
		return nil, fmt.Errorf("profile: %w", err)
	}

	switch {
	case a.Config.Local != nil:
		return &local.Args{
			Profile: prof,
			Spec:    *a.Config.Local,
		}, nil

	case a.Config.Cloud != nil && a.Config.Cloud.DigitalOcean != nil:
		return &digitalocean.Args{
			Profile:   prof,
			Cloud:     a.Cloud,
			Config:    a.Config.Cloud.DigitalOcean,
			Namespace: a.Namespace,
			Name:      name,
		}, nil
	}

	return nil, fmt.Errorf("dispatch: no backend selected (validate should have caught this)")
}

func (a *Args) validate() error {
	if a.Namespace == "" {
		return fmt.Errorf("Namespace is required")
	}
	if err := a.Env.Validate(); err != nil {
		return fmt.Errorf("invalid env: %w", err)
	}
	if err := a.Config.Validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if a.Env.IsLocal() && a.Config.Local == nil {
		return fmt.Errorf("local environments require the local backend")
	}
	if !a.Env.IsLocal() && a.Config.Cloud == nil {
		return fmt.Errorf("non-local environments require a cloud backend")
	}

	if a.Config.Cloud != nil && a.Config.Cloud.DigitalOcean != nil &&
		a.Cloud.DigitalOcean == nil {
		return fmt.Errorf("Cloud.DigitalOcean is required when config.cloud.digitalocean is set")
	}

	return nil
}
