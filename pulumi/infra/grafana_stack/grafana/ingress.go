package grafana

import (
	"fmt"
	"net/url"

	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	networkingv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/util/annotations"
	"github.com/milagre/zote/pulumi/util/tokens"
)

const tlsSecretName = "grafana-tls"

// registerIngresses provisions the public nginx (and, locally, cloudflare-tunnel)
// Ingress for Grafana. It returns the endpoint callers should use to reach it
// (the primary public HTTPS host when an Ingress was created, otherwise inCluster)
// and the created Ingress resources so callers can sequence work after them.
func registerIngresses(ctx *pulumi.Context, name string, args *Args, g *Grafana, parent pulumi.Resource, inCluster url.URL) (url.URL, []pulumi.Resource, error) {
	if len(args.PublicDomains) == 0 {
		return inCluster, nil, nil
	}

	hosts := ingressHosts(name, args.Namespace, args.PublicDomains)
	var ingresses []pulumi.Resource

	if class := pulumi.StringPtrFromPtr(publicIngressClassName(args.Cluster)); class != nil {
		ing, err := registerPublicIngress(ctx, name, args, g, parent, class, hosts)
		if err != nil {
			return url.URL{}, nil, err
		}
		if ing != nil {
			ingresses = append(ingresses, ing)
		}
	}

	if args.Env.IsLocal() {
		if class := pulumi.StringPtrFromPtr(tunnelIngressClassName(args.Cluster)); class != nil {
			ing, err := registerTunnelIngress(ctx, name, args, g, parent, class, hosts)
			if err != nil {
				return url.URL{}, nil, err
			}

			ingresses = append(ingresses, ing)
		}
	}

	if len(ingresses) == 0 {
		return inCluster, nil, nil
	}

	return publicURL(hosts[0]), ingresses, nil
}

// registerPublicIngress returns a nil Ingress (without error) when a non-local
// cluster has no ClusterIssuer, since TLS cannot be provisioned.
func registerPublicIngress(
	ctx *pulumi.Context,
	name string,
	args *Args,
	g *Grafana,
	parent pulumi.Resource,
	ingressClass pulumi.StringPtrInput,
	hosts []string,
) (*networkingv1.Ingress, error) {
	className := publicIngressClassName(args.Cluster)
	ann := pulumi.StringMap{
		annotations.WaitForKey:        pulumi.String(annotations.WaitForValueImmediate),
		"kubernetes.io/ingress.class": pulumi.String(*className),
	}

	spec := &networkingv1.IngressSpecArgs{
		IngressClassName: ingressClass,
		Rules:            hostRules(name, hosts),
	}
	if !args.Env.IsLocal() {
		issuer := clusterIssuerName(args.Cluster)
		if issuer == nil {
			return nil, nil
		}

		ann["cert-manager.io/cluster-issuer"] = pulumi.String(*issuer)

		tlsHosts := make(pulumi.StringArray, 0, len(hosts))
		for _, h := range hosts {
			tlsHosts = append(tlsHosts, pulumi.String(h))
		}
		spec.Tls = networkingv1.IngressTLSArray{
			&networkingv1.IngressTLSArgs{
				Hosts:      tlsHosts,
				SecretName: pulumi.String(tlsSecretName),
			},
		}
	}

	ing, err := networkingv1.NewIngress(ctx, tokens.Qualify(args.Namespace, "grafana-nginx"), &networkingv1.IngressArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String("grafana-nginx"),
			Namespace:   pulumi.String(args.Namespace),
			Annotations: ann,
		},
		Spec: spec,
	}, pulumi.Parent(parent), pulumi.DependsOn(ingressDeps(args, g)))
	if err != nil {
		return nil, fmt.Errorf("public ingress: %w", err)
	}

	return ing, nil
}

func registerTunnelIngress(
	ctx *pulumi.Context,
	name string,
	args *Args,
	g *Grafana,
	parent pulumi.Resource,
	ingressClass pulumi.StringPtrInput,
	hosts []string,
) (*networkingv1.Ingress, error) {
	className := tunnelIngressClassName(args.Cluster)

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
			IngressClassName: ingressClass,
			Rules:            hostRules(name, hosts),
		},
	}, pulumi.Parent(parent), pulumi.DependsOn(ingressDeps(args, g)))
	if err != nil {
		return nil, fmt.Errorf("tunnel ingress: %w", err)
	}

	return ing, nil
}

func ingressDeps(args *Args, g *Grafana) []pulumi.Resource {
	deps := append([]pulumi.Resource{}, args.IngressDeps...)
	deps = append(deps, g.Helm.Release)

	return deps
}

// ingressHosts returns the fully qualified public hostnames (<name>.<namespace>.<domain>)
// Grafana is served at, one per public domain. The first is the primary.
func ingressHosts(name, namespace string, publicDomains []string) []string {
	out := make([]string, 0, len(publicDomains))
	for _, d := range publicDomains {
		out = append(out, fmt.Sprintf("%s.%s.%s", name, namespace, d))
	}

	return out
}

// publicURL is the external HTTPS endpoint for host; ingress and tunnel both terminate TLS.
func publicURL(host string) url.URL {
	return url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/",
	}
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

func publicIngressClassName(cluster *infra.Cluster) *string {
	if cluster == nil {
		return nil
	}

	return cluster.PublicIngressClassName
}

func tunnelIngressClassName(cluster *infra.Cluster) *string {
	if cluster == nil {
		return nil
	}

	return cluster.TunnelIngressClassName
}

func clusterIssuerName(cluster *infra.Cluster) *string {
	if cluster == nil {
		return nil
	}

	return cluster.ClusterIssuerName
}
