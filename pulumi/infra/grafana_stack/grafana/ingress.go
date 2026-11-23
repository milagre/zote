package grafana

import (
	"fmt"

	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	networkingv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/util/annotations"
	"github.com/milagre/zote/pulumi/util/tokens"
)

// registerTunnelIngress serves Grafana's public hostnames through the
// cloudflare-tunnel ingress class in local environments. It returns nil when
// there is no tunnel or nothing to serve.
func registerTunnelIngress(ctx *pulumi.Context, name string, args *Args, g *Grafana, parent pulumi.Resource) (pulumi.Resource, error) {
	className := tunnelIngressClassName(args.Cluster)
	if !args.Env.IsLocal() || className == nil || len(args.PublicDomains) == 0 {
		return nil, nil
	}

	hosts := publicHosts(name, args.Namespace, args.PublicDomains)

	ing, err := networkingv1.NewIngress(ctx, tokens.Qualify(args.Namespace, "grafana-tunnel"), &networkingv1.IngressArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("grafana-tunnel"),
			Namespace: pulumi.String(args.Namespace),
			Annotations: pulumi.StringMap{
				annotations.WaitForKey:        pulumi.String(annotations.WaitForValueImmediate),
				"kubernetes.io/ingress.class": pulumi.String(*className),
			},
		},
		Spec: &networkingv1.IngressSpecArgs{
			IngressClassName: pulumi.String(*className),
			Rules:            hostRules(name, hosts),
		},
	}, pulumi.Parent(parent), pulumi.DependsOn(publicDeps(args, g)))
	if err != nil {
		return nil, fmt.Errorf("tunnel ingress: %w", err)
	}

	return ing, nil
}

func hostRules(svcName string, hosts []string) networkingv1.IngressRuleArray {
	rules := make(networkingv1.IngressRuleArray, 0, len(hosts))
	for _, h := range hosts {
		rules = append(rules, hostRule(svcName, h))
	}

	return rules
}

func hostRule(svcName, host string) *networkingv1.IngressRuleArgs {
	pathType := "Prefix"

	return &networkingv1.IngressRuleArgs{
		Host: pulumi.String(host),
		Http: &networkingv1.HTTPIngressRuleValueArgs{
			Paths: networkingv1.HTTPIngressPathArray{
				&networkingv1.HTTPIngressPathArgs{
					Path:     pulumi.String("/"),
					PathType: pulumi.String(pathType),
					Backend: &networkingv1.IngressBackendArgs{
						Service: &networkingv1.IngressServiceBackendArgs{
							Name: pulumi.String(svcName),
							Port: &networkingv1.ServiceBackendPortArgs{
								Number: pulumi.Int(80),
							},
						},
					},
				},
			},
		},
	}
}

func tunnelIngressClassName(cluster *infra.Cluster) *string {
	if cluster == nil {
		return nil
	}

	return cluster.TunnelIngressClassName
}
