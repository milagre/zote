// Package nginx_ingress installs ingress-nginx with a product-neutral
// set of production defaults: autoscaling enabled, pod anti-affinity that
// spreads controllers across nodes, and — on clusters without a
// LoadBalancer provisioner (local minikube/kind) — a NodePort Service so
// the chart doesn't sit Pending forever.
//
// Sizing (CPU/memory requests and limits, and the replica count band the
// autoscaler works against) is caller-supplied via a validated
// profile.Profile. Everything else chart-specific is owned here;
// extending the schema (bespoke annotations, for example) is a deliberate
// change to this package, not an ad-hoc override at each call site.
package nginx_ingress

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/tokens"
	"github.com/milagre/zote/pulumi/profile"
)

var spec = helm.ChartSpec{
	TypeToken:      tokens.Token("infra", "NginxIngress"),
	Chart:          "ingress-nginx",
	Repository:     "https://kubernetes.github.io/ingress-nginx",
	DefaultVersion: "4.7.2",
}

// Args are the caller-supplied inputs. Values is intentionally absent:
// the chart's value tree is fully owned by this package so every
// cluster in the fleet gets the same battle-tested defaults for
// everything except sizing, which Profile drives.
type Args struct {
	// Namespace is the target namespace. Required.
	Namespace string

	// Env determines cluster-type-dependent behavior — specifically
	// whether the controller Service has to be NodePort (local) or
	// can accept the chart default (LoadBalancer, remote).
	Env env.Env

	// Profile sizes the controller pods. CPUCores and MemMB become
	// resources.requests/limits on the pod spec; Num (when non-nil)
	// sets autoscaling.minReplicas / maxReplicas and the initial
	// replicaCount. When Num is nil, the upstream chart's default
	// replica/autoscaler band is used.
	Profile profile.Profile

	// Version overrides DefaultVersion. Optional.
	Version *string
}

// NginxIngress is the installed chart, wrapped as a ComponentResource
// so callers can express pulumi.DependsOn against it when their own
// ingresses need to come up after the controller.
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

// values renders the chart Values tree. The shape tracks the upstream
// ingress-nginx chart's Values schema; see
// https://github.com/kubernetes/ingress-nginx/tree/main/charts/ingress-nginx
// for the authoritative reference.
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
