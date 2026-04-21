// Package prometheus installs the upstream kube-prometheus-stack Helm
// chart with a fleet-wide opinionated default: the Prometheus CR is
// configured to pick up *every* PodMonitor in the cluster, in any
// namespace, regardless of labels. That inverts the upstream default
// (which only discovers PodMonitors matching the release's own
// labels) and lets application namespaces drop a PodMonitor next to
// their pods without coordinating label selectors with this install.
package prometheus

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/tokens"
)

const servicePort = 9090

var spec = helm.ChartSpec{
	TypeToken:      tokens.Token("infra", "Prometheus"),
	Chart:          "kube-prometheus-stack",
	Repository:     "https://prometheus-community.github.io/helm-charts",
	DefaultVersion: "80.6.0",
}

// Args are the caller-supplied inputs.
type Args struct {
	// Namespace is the target namespace. Required.
	Namespace string

	// Version overrides DefaultVersion. Optional.
	Version *string
}

// Service identifies the in-cluster Service that callers use to scrape
// or query prometheus: the chart release name, its namespace, and the
// well-known prometheus HTTP port.
type Service struct {
	Name      pulumi.StringOutput
	Namespace pulumi.StringOutput
	Port      pulumi.IntOutput
}

// Prometheus is the component resource. Service locates the chart's
// query endpoint so dependent components (dashboards, scrapers) can
// address it without rediscovering the coordinates.
type Prometheus struct {
	helm.ChartComponent

	Service Service
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Prometheus, error) {
	if args == nil {
		return nil, fmt.Errorf("prometheus: args is required")
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("prometheus: Namespace is required")
	}

	comp := &Prometheus{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   args.Version,
		Values: helm.Values(map[string]any{
			"prometheus": map[string]any{
				"prometheusSpec": map[string]any{
					"podMonitorNamespaceSelector": map[string]any{},
					"podMonitorSelector": map[string]any{
						"matchLabels": map[string]any{},
					},
				},
			},
		}),
	}, &comp.ChartComponent, opts...); err != nil {
		return nil, err
	}

	comp.Service = Service{
		Name:      pulumi.String(name).ToStringOutput(),
		Namespace: pulumi.String(args.Namespace).ToStringOutput(),
		Port:      pulumi.Int(servicePort).ToIntOutput(),
	}

	return comp, nil
}
