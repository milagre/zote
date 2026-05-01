// Package nginx_ingress installs ingress-nginx with fleet defaults (HPA, spread); local env uses NodePort. Sizing via [profile.Profile] only.
package nginx_ingress

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/profile"
	"github.com/milagre/zote/pulumi/tokens"
)

var spec = helm.ChartSpec{
	TypeToken:      tokens.Token("infra", "NginxIngress"),
	Chart:          "ingress-nginx",
	Repository:     "https://kubernetes.github.io/ingress-nginx",
	DefaultVersion: "4.7.2",
}

type Args struct {
	Namespace string
	Env       env.Env
	Profile   profile.Profile
	Version   *string
}

type NginxIngress struct {
	helm.ChartComponent
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*NginxIngress, error) {
	if args == nil {
		return nil, fmt.Errorf("nginx_ingress: args is required")
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("nginx_ingress: Namespace is required")
	}

	comp := &NginxIngress{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   args.Version,
		Values:    values(args.Env, args.Profile),
	}, &comp.ChartComponent, opts...); err != nil {
		return nil, err
	}

	return comp, nil
}

func values(e env.Env, p profile.Profile) pulumi.Map {
	autoscaling := map[string]any{
		"enabled":                           true,
		"minReplicas":                       3,
		"maxReplicas":                       6,
		"targetCPUUtilizationPercentage":    75,
		"targetMemoryUtilizationPercentage": 75,
	}

	controller := map[string]any{
		"autoscaling": autoscaling,
		"resources": map[string]any{
			"requests": map[string]any{
				"cpu":    p.MinCoresMilli(),
				"memory": p.MinMemMiB(),
			},
			"limits": map[string]any{
				"cpu":    p.MaxCoresMilli(),
				"memory": p.MaxMemMiB(),
			},
		},
		"affinity": map[string]any{
			"podAntiAffinity": map[string]any{
				"preferredDuringSchedulingIgnoredDuringExecution": []any{
					map[string]any{
						"weight": 100,
						"podAffinityTerm": map[string]any{
							"labelSelector": map[string]any{
								"matchExpressions": []any{
									map[string]any{"key": "app.kubernetes.io/name", "operator": "In", "values": []any{"ingress-nginx"}},
									map[string]any{"key": "app.kubernetes.io/instance", "operator": "In", "values": []any{"ingress-nginx"}},
									map[string]any{"key": "app.kubernetes.io/component", "operator": "In", "values": []any{"controller"}},
								},
							},
							"topologyKey": "kubernetes.io/hostname",
						},
					},
				},
			},
		},
	}

	if p.Num != nil {
		controller["replicaCount"] = p.Num.Min
		autoscaling["minReplicas"] = p.Num.Min
		autoscaling["maxReplicas"] = p.Num.Max
	}

	if e.IsLocal() {
		controller["service"] = map[string]any{"type": "NodePort"}
	}

	return helm.Values(map[string]any{"controller": controller})
}
