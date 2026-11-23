// Package httproute serves a Service through the cluster's shared Gateway.
package httproute

import (
	"fmt"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/util/annotations"
)

// hsts tells browsers to reach the hostname, and its subdomains, only over
// HTTPS for a year.
const hsts = "max-age=31536000; includeSubDomains"

type Args struct {
	Env       env.Env
	Namespace string
	Name      string

	Hostnames []string
	Service   pulumi.StringInput
	Port      int

	Cluster *infra.Cluster
}

// Register creates the HTTPRoute serving Hostnames from Service:Port.
// Outside local environments it also creates a ListenerSet on the Gateway
// with an HTTPS listener per hostname, whose certificate comes from the
// cluster's ClusterIssuer, and the route attaches to those listeners. Locally
// the route attaches to the Gateway's plain HTTP listener. Register creates
// nothing, and returns nil, when the cluster has no Gateway or TLS has no
// issuer.
func Register(ctx *pulumi.Context, name string, args Args, opts ...pulumi.ResourceOption) (*apiextensions.CustomResource, error) {
	if args.Cluster == nil || args.Cluster.Gateway == nil || len(args.Hostnames) == 0 {
		return nil, nil
	}

	gw := args.Cluster.Gateway
	tls := !args.Env.IsLocal()
	if tls && args.Cluster.ClusterIssuerName == nil {
		return nil, nil
	}

	opts = append(opts, pulumi.DependsOn(gw.Ready))

	parentRefs := pulumi.Array{
		pulumi.Map{
			"group":       pulumi.String("gateway.networking.k8s.io"),
			"kind":        pulumi.String("Gateway"),
			"name":        pulumi.String(gw.Name),
			"namespace":   pulumi.String(gw.Namespace),
			"sectionName": pulumi.String(infra.GatewayHTTPListener),
		},
	}
	if tls {
		set, err := registerListenerSet(ctx, name, args, opts)
		if err != nil {
			return nil, err
		}

		parentRefs = make(pulumi.Array, 0, len(args.Hostnames))
		for _, h := range args.Hostnames {
			parentRefs = append(parentRefs, pulumi.Map{
				"group":       pulumi.String("gateway.networking.k8s.io"),
				"kind":        pulumi.String("ListenerSet"),
				"name":        pulumi.String(args.Name),
				"sectionName": pulumi.String(infra.GatewayListenerName(h)),
			})
		}

		opts = append(opts, pulumi.DependsOn([]pulumi.Resource{set}))
	}

	hostnames := make(pulumi.StringArray, 0, len(args.Hostnames))
	for _, h := range args.Hostnames {
		hostnames = append(hostnames, pulumi.String(h))
	}

	rule := pulumi.Map{
		"matches": pulumi.Array{
			pulumi.Map{"path": pulumi.Map{
				"type":  pulumi.String("PathPrefix"),
				"value": pulumi.String("/"),
			}},
		},
		"backendRefs": pulumi.Array{
			pulumi.Map{
				"name": args.Service,
				"port": pulumi.Int(args.Port),
			},
		},
	}
	if tls {
		rule["filters"] = pulumi.Array{
			pulumi.Map{
				"type": pulumi.String("ResponseHeaderModifier"),
				"responseHeaderModifier": pulumi.Map{
					"set": pulumi.Array{
						pulumi.Map{
							"name":  pulumi.String("Strict-Transport-Security"),
							"value": pulumi.String(hsts),
						},
					},
				},
			},
		}
	}

	route, err := apiextensions.NewCustomResource(ctx, name, &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("gateway.networking.k8s.io/v1"),
		Kind:       pulumi.String("HTTPRoute"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(args.Name),
			Namespace: pulumi.String(args.Namespace),
			Annotations: pulumi.StringMap{
				annotations.WaitForKey: pulumi.String(annotations.WaitForValueImmediate),
			},
		},
		OtherFields: map[string]any{
			"spec": pulumi.Map{
				"parentRefs": parentRefs,
				"hostnames":  hostnames,
				"rules":      pulumi.Array{rule},
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("http route: %w", err)
	}

	return route, nil
}

// registerListenerSet attaches an HTTPS listener per hostname to the Gateway,
// each with a certificate Secret of its own in the route's namespace.
func registerListenerSet(ctx *pulumi.Context, name string, args Args, opts []pulumi.ResourceOption) (pulumi.Resource, error) {
	gw := args.Cluster.Gateway

	listeners := make(pulumi.Array, 0, len(args.Hostnames))
	for _, h := range args.Hostnames {
		listeners = append(listeners, pulumi.Map{
			"name":     pulumi.String(infra.GatewayListenerName(h)),
			"hostname": pulumi.String(h),
			"port":     pulumi.Int(443),
			"protocol": pulumi.String("HTTPS"),
			"tls": pulumi.Map{
				"mode": pulumi.String("Terminate"),
				"certificateRefs": pulumi.Array{
					pulumi.Map{
						"kind": pulumi.String("Secret"),
						"name": pulumi.String(h + "-tls"),
					},
				},
			},
			"allowedRoutes": pulumi.Map{
				"namespaces": pulumi.Map{"from": pulumi.String("Same")},
			},
		})
	}

	set, err := apiextensions.NewCustomResource(ctx, name+"-listeners", &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("gateway.networking.k8s.io/v1"),
		Kind:       pulumi.String("ListenerSet"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(args.Name),
			Namespace: pulumi.String(args.Namespace),
			Annotations: pulumi.StringMap{
				annotations.WaitForKey:           pulumi.String(annotations.WaitForValueImmediate),
				"cert-manager.io/cluster-issuer": pulumi.String(*args.Cluster.ClusterIssuerName),
			},
		},
		OtherFields: map[string]any{
			"spec": pulumi.Map{
				"parentRef": pulumi.Map{
					"group":     pulumi.String("gateway.networking.k8s.io"),
					"kind":      pulumi.String("Gateway"),
					"name":      pulumi.String(gw.Name),
					"namespace": pulumi.String(gw.Namespace),
				},
				"listeners": listeners,
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("listener set: %w", err)
	}

	return set, nil
}
