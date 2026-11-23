// Package cert_manager installs cert-manager plus a letsencrypt-http01 ClusterIssuer.
// With a Gateway on the cluster it also issues certificates for the listeners
// attached to it, so it installs after the Gateway API CRDs.
package cert_manager

import (
	"fmt"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/util/tokens"
)

const ClusterIssuerName = "letsencrypt-http01"

var spec = helm.ChartSpec{
	TypeToken:      tokens.Token("infra", "CertManager"),
	Chart:          "cert-manager",
	Repository:     "https://charts.jetstack.io",
	DefaultVersion: "1.21.3",
}

type Args struct {
	Namespace      string
	AcmeEmail      string
	IngressClasses []string

	Config Config

	// Cluster registers deployed capabilities when non-nil.
	Cluster *infra.Cluster
}

type CertManager struct {
	helm.ChartComponent

	ClusterIssuer *apiextensions.CustomResource
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*CertManager, error) {
	if args == nil {
		return nil, fmt.Errorf("cert_manager: args is required")
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("cert_manager: Namespace is required")
	}
	if args.AcmeEmail == "" {
		return nil, fmt.Errorf("cert_manager: AcmeEmail is required")
	}
	if err := args.Config.Validate(); err != nil {
		return nil, fmt.Errorf("cert_manager: config: %w", err)
	}

	var deps []pulumi.Resource
	if args.Cluster != nil && args.Cluster.Gateway != nil {
		deps = args.Cluster.Gateway.Ready
	}

	comp := &CertManager{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   helm.OptionalChartVersion(args.Config.Version),
		Values:    values(args),
		DependsOn: deps,
	}, &comp.ChartComponent, opts...); err != nil {
		return nil, err
	}

	issuer, err := apiextensions.NewCustomResource(ctx, ClusterIssuerName, &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("cert-manager.io/v1"),
		Kind:       pulumi.String("ClusterIssuer"),
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String(ClusterIssuerName),
		},
		OtherFields: map[string]any{
			"spec": pulumi.Map{
				"acme": pulumi.Map{
					"email":  pulumi.String(args.AcmeEmail),
					"server": pulumi.String("https://acme-v02.api.letsencrypt.org/directory"),
					"privateKeySecretRef": pulumi.Map{
						"name": pulumi.String(ClusterIssuerName),
					},
					"solvers": http01Solvers(args),
				},
			},
		},
	}, pulumi.Parent(comp), pulumi.DependsOn([]pulumi.Resource{comp.Release}))
	if err != nil {
		return nil, fmt.Errorf("cert_manager: cluster issuer: %w", err)
	}

	comp.ClusterIssuer = issuer

	if args.Cluster != nil {
		args.Cluster.SetClusterIssuer(ClusterIssuerName)
	}

	return comp, nil
}

// values enables cert-manager's Gateway support only with a Gateway on the
// cluster: it fails to start without the Gateway API CRDs. Certificates for
// ListenerSets sit behind an alpha feature gate.
func values(args *Args) pulumi.Map {
	v := map[string]any{
		"installCRDs": true,
	}
	if args.Cluster != nil && args.Cluster.Gateway != nil {
		v["config"] = map[string]any{
			"gatewayAPI":   map[string]any{"enabled": true},
			"featureGates": map[string]any{"ListenerSets": true},
		}
	}

	return helm.Values(v)
}

// http01Solvers lists the ingress classes, then the cluster's Gateway:
// cert-manager answers through the first, so the Gateway's is used only once
// there are no ingress classes.
func http01Solvers(args *Args) pulumi.Array {
	out := make(pulumi.Array, 0, len(args.IngressClasses)+1)
	for _, c := range args.IngressClasses {
		out = append(out, pulumi.Map{
			"http01": pulumi.Map{
				"ingress": pulumi.Map{
					"class": pulumi.String(c),
				},
			},
		})
	}

	if args.Cluster != nil && args.Cluster.Gateway != nil {
		gw := args.Cluster.Gateway
		out = append(out, pulumi.Map{
			"http01": pulumi.Map{
				"gatewayHTTPRoute": pulumi.Map{
					"parentRefs": pulumi.Array{
						pulumi.Map{
							"name":        pulumi.String(gw.Name),
							"namespace":   pulumi.String(gw.Namespace),
							"kind":        pulumi.String("Gateway"),
							"sectionName": pulumi.String(infra.GatewayHTTPListener),
						},
					},
				},
			},
		})
	}

	return out
}
