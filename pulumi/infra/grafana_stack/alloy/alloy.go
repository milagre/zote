// Package alloy installs upstream grafana alloy
package alloy

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/util/tokens"
)

var spec = helm.ChartSpec{
	TypeToken:      tokens.Token("infra", "Alloy"),
	Chart:          "alloy",
	Repository:     "https://grafana.github.io/helm-charts",
	DefaultVersion: "1.12.1",
}

type Args struct {
	Namespace string

	Config Config

	River pulumi.StringInput
}

type Alloy struct {
	helm.ChartComponent
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Alloy, error) {
	if err := validateArgs(args); err != nil {
		return nil, fmt.Errorf("alloy: %w", err)
	}

	values, err := chartValues(args.River, args.Config)
	if err != nil {
		return nil, fmt.Errorf("alloy: values: %w", err)
	}

	comp := &Alloy{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   helm.OptionalChartVersion(args.Config.Version),
		Values:    values,
	}, &comp.ChartComponent, opts...); err != nil {
		return nil, err
	}

	return comp, nil
}
