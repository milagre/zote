// Package keda installs upstream KEDA (Kubernetes Event-driven Autoscaling).
package keda

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/util/tokens"
)

var spec = helm.ChartSpec{
	TypeToken:      tokens.Token("infra", "Keda"),
	Chart:          "keda",
	Repository:     "https://kedacore.github.io/charts",
	DefaultVersion: "2.19.0",
}

type Args struct {
	Namespace string

	Config Config

	// Cluster registers deployed capabilities when non-nil.
	Cluster *infra.Cluster
}

type Keda struct {
	helm.ChartComponent
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Keda, error) {
	if args == nil {
		return nil, fmt.Errorf("keda: args is required")
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("keda: Namespace is required")
	}
	if err := args.Config.Validate(); err != nil {
		return nil, fmt.Errorf("keda: config: %w", err)
	}

	comp := &Keda{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   helm.OptionalChartVersion(args.Config.Version),
	}, &comp.ChartComponent, opts...); err != nil {
		return nil, err
	}

	if args.Cluster != nil {
		args.Cluster.HasKeda = true
	}

	return comp, nil
}
