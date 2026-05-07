// Package metrics_server installs upstream metrics-server (default chart values).
package metrics_server

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/tokens"
)

var spec = helm.ChartSpec{
	TypeToken:      tokens.Token("infra", "MetricsServer"),
	Chart:          "metrics-server",
	Repository:     "https://kubernetes-sigs.github.io/metrics-server",
	DefaultVersion: "3.11.0",
}

type Args struct {
	Namespace string

	Config Config
}

type MetricsServer struct {
	helm.ChartComponent
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*MetricsServer, error) {
	if args == nil {
		return nil, fmt.Errorf("metrics_server: args is required")
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("metrics_server: Namespace is required")
	}
	if err := args.Config.Validate(); err != nil {
		return nil, fmt.Errorf("metrics_server: config: %w", err)
	}

	comp := &MetricsServer{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   helm.OptionalChartVersion(args.Config.Version),
	}, &comp.ChartComponent, opts...); err != nil {
		return nil, err
	}

	return comp, nil
}
