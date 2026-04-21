// Package grafana installs the upstream grafana Helm chart with the
// chart's default value tree.
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

// Args are the caller-supplied inputs. The chart's value tree is left at
// the upstream default; callers compose access-layer resources (ingress,
// grafana API objects) alongside the component.
type Args struct {
	// Namespace is the target namespace. Required.
	Namespace string

	// Version overrides DefaultVersion. Optional.
	Version *string
}

// Grafana is the installed chart, wrapped as a ComponentResource so
// callers can express pulumi.DependsOn against it when staging
// downstream resources (datasource wiring, ingress, etc.).
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
