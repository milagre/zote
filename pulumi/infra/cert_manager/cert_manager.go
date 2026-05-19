// Package cert_manager installs cert-manager plus a letsencrypt-http01 ClusterIssuer.
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
	DefaultVersion: "1.13.1",
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

	comp := &CertManager{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   helm.OptionalChartVersion(args.Config.Version),
		Values: helm.Values(map[string]any{
			"installCRDs": true,
		}),
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
					"solvers": http01IngressSolvers(args.IngressClasses),
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

func http01IngressSolvers(classes []string) pulumi.Array {
	out := make(pulumi.Array, 0, len(classes))
	for _, c := range classes {
		out = append(out, pulumi.Map{
			"http01": pulumi.Map{
				"ingress": pulumi.Map{
					"class": pulumi.String(c),
				},
			},
		})
	}

	return out
}
