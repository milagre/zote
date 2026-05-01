// Package prometheus installs kube-prometheus-stack; Prometheus scrapes all PodMonitors cluster-wide (not release-scoped).
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

type Args struct {
	Namespace string
	Version   *string
}

type Service struct {
	Name      pulumi.StringOutput
	Namespace pulumi.StringOutput
	Port      pulumi.IntOutput
}

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
