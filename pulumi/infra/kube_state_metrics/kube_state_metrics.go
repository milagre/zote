// Package kube_state_metrics installs upstream kube-state-metrics, annotated
// for the Grafana stack's pod scrape so object state (e.g. a Deployment's
// available replicas) lands in Mimir under the object's own namespace and name.
package kube_state_metrics

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/infra/grafana_stack"
	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/util/tokens"
)

// metricsPort is the chart's default port for object metrics.
const metricsPort = "8080"

var spec = helm.ChartSpec{
	TypeToken:      tokens.Token("infra", "KubeStateMetrics"),
	Chart:          "kube-state-metrics",
	Repository:     "https://prometheus-community.github.io/helm-charts",
	DefaultVersion: "8.5.0",
}

type Args struct {
	Namespace string

	Config Config
}

type KubeStateMetrics struct {
	helm.ChartComponent
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*KubeStateMetrics, error) {
	if args == nil {
		return nil, fmt.Errorf("kube_state_metrics: args is required")
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("kube_state_metrics: Namespace is required")
	}
	if err := args.Config.Validate(); err != nil {
		return nil, fmt.Errorf("kube_state_metrics: config: %w", err)
	}

	comp := &KubeStateMetrics{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   helm.OptionalChartVersion(args.Config.Version),
		Values:    chartValues(),
	}, &comp.ChartComponent, opts...); err != nil {
		return nil, fmt.Errorf("kube_state_metrics: %w", err)
	}

	return comp, nil
}

func chartValues() pulumi.Map {
	return pulumi.Map{
		// The chart's own prometheusScrape annotates the Service, which the pod
		// scrape never sees.
		"prometheusScrape": pulumi.Bool(false),
		"podAnnotations": pulumi.StringMap{
			"prometheus.io/scrape":              pulumi.String("true"),
			"prometheus.io/port":                pulumi.String(metricsPort),
			grafana_stack.HonorLabelsAnnotation: pulumi.String("true"),
		},
	}
}
