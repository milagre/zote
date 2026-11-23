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

// Register creates the HTTPRoute serving Hostnames from Service:Port and
// records them as needing Gateway listeners. Outside local environments the
// route attaches to each hostname's HTTPS listener, which needs a
// ClusterIssuer for its certificate; locally it attaches to the plain HTTP
// listener. Register creates nothing, and returns nil, when the cluster has
// no Gateway or a TLS route has no issuer.
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

	sections := []string{infra.GatewayHTTPListener}
	if tls {
		sections = make([]string, 0, len(args.Hostnames))
		for _, h := range args.Hostnames {
			sections = append(sections, infra.GatewayListenerName(h))
		}
	}

	parentRefs := make(pulumi.Array, 0, len(sections))
	for _, s := range sections {
		parentRefs = append(parentRefs, pulumi.Map{
			"group":       pulumi.String("gateway.networking.k8s.io"),
			"kind":        pulumi.String("Gateway"),
			"name":        pulumi.String(gw.Name),
			"namespace":   pulumi.String(gw.Namespace),
			"sectionName": pulumi.String(s),
		})
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

	gw.AddHostnames(args.Hostnames...)

	return route, nil
}
