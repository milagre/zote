// Package alloy installs upstream grafana alloy
package alloy

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/tokens"
)

var spec = helm.ChartSpec{
	TypeToken:      tokens.Token("infra", "Alloy"),
	Chart:          "alloy",
	Repository:     "https://grafana.github.io/helm-charts",
	DefaultVersion: "1.8.1",
}

type Args struct {
	Namespace string

	Config Config
}

type Alloy struct {
	helm.ChartComponent
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Alloy, error) {
	if args == nil {
		return nil, fmt.Errorf("alloy: args is required")
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("alloy: Namespace is required")
	}
	if err := args.Config.Validate(); err != nil {
		return nil, fmt.Errorf("alloy: config: %w", err)
	}

	comp := &Alloy{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   helm.OptionalChartVersion(args.Config.Version),
	}, &comp.ChartComponent, opts...); err != nil {
		return nil, err
	}

	return comp, nil
}
