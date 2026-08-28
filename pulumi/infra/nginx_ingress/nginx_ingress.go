// Package nginx_ingress installs ingress-nginx with fleet defaults (HPA, spread); local env uses NodePort. Sizing via [profile.Profile] only.
package nginx_ingress

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/util/profile"
	"github.com/milagre/zote/pulumi/util/tokens"
)

var spec = helm.ChartSpec{
	TypeToken:      tokens.Token("infra", "NginxIngress"),
	Chart:          "ingress-nginx",
	Repository:     "https://kubernetes.github.io/ingress-nginx",
	DefaultVersion: "4.15.1",
}

type Args struct {
	Namespace string
	Env       env.Env

	IngressClassName string

	Config Config

	// Cluster registers deployed capabilities when non-nil.
	Cluster *infra.Cluster
}

type NginxIngress struct {
	helm.ChartComponent

	IngressClassName pulumi.StringOutput
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*NginxIngress, error) {
	if args == nil {
		return nil, fmt.Errorf("nginx_ingress: args is required")
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("nginx_ingress: Namespace is required")
	}
	prof, err := args.Config.ResourceProfile()
	if err != nil {
		return nil, fmt.Errorf("nginx_ingress: config: %w", err)
	}

	ingressClass := ingressClassName(args.IngressClassName)

	comp := &NginxIngress{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   helm.OptionalChartVersion(args.Config.Version),
		Values:    values(args.Env, prof, ingressClass),
	}, &comp.ChartComponent, opts...); err != nil {
		return nil, err
	}

	comp.IngressClassName = pulumi.String(ingressClass).ToStringOutput()

	args.Cluster.SetPublicIngressClass(ingressClass)
	args.Cluster.SetPublicIngressService(name+"-controller", args.Namespace)

	return comp, nil
}

func values(e env.Env, p profile.Profile, ingressClass string) pulumi.Map {
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

	controller["ingressClassResource"] = map[string]any{
		"name":    ingressClass,
		"enabled": true,
	}

	return helm.Values(map[string]any{"controller": controller})
}

// ingressClassName returns s trimmed, or [DefaultIngressClass] when empty.
func ingressClassName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return DefaultIngressClass
	}

	return s
}
