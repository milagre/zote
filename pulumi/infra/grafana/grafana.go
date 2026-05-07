// Package grafana installs upstream grafana
package grafana

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/tokens"
)

var spec = helm.ChartSpec{
	TypeToken:      tokens.Token("infra", "Grafana"),
	Chart:          "grafana",
	Repository:     "https://grafana-community.github.io/helm-charts",
	DefaultVersion: "12.1.1",
}

type Args struct {
	Namespace string
	Version   *string
}

type Grafana struct {
	helm.ChartComponent
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Grafana, error) {
	if args == nil {
		return nil, fmt.Errorf("grafana: args is required")
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("grafana: Namespace is required")
	}

	comp := &Grafana{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   args.Version,
	}, &comp.ChartComponent, opts...); err != nil {
		return nil, err
	}

	return comp, nil
}
