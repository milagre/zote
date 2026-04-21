// Package metrics_server installs the upstream kubernetes-sigs
// metrics-server Helm chart with the chart's default value tree.
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

// Args are the caller-supplied inputs. No chart values are exposed; the
// upstream default tree is installed as-is.
type Args struct {
	// Namespace is the target namespace. Required.
	Namespace string

	// Version overrides DefaultVersion. Optional.
	Version *string
}

// MetricsServer is the installed chart, wrapped as a ComponentResource
// so callers can express pulumi.DependsOn against it when a downstream
// resource (an HPA, for example) needs the metrics API to be online.
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

	comp := &MetricsServer{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   args.Version,
	}, &comp.ChartComponent, opts...); err != nil {
		return nil, err
	}

	return comp, nil
}
