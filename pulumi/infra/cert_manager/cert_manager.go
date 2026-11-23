// Package cert_manager installs cert-manager plus a letsencrypt-http01
// ClusterIssuer, which issues the certificates for the cluster Gateway's
// listeners and answers HTTP-01 challenges through the Gateway. The Gateway
// must be recorded on the Cluster first, since cert-manager starts only once
// the Gateway API CRDs are installed.
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
	Namespace string
	AcmeEmail string

	Config Config

	// Cluster supplies the Gateway, and records the ClusterIssuer.
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
	if args.Cluster == nil || args.Cluster.Gateway == nil {
		return nil, fmt.Errorf("cert_manager: Cluster must have a Gateway")
	}

	gw := args.Cluster.Gateway

	comp := &CertManager{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   helm.OptionalChartVersion(args.Config.Version),
		Values:    values(),
		DependsOn: gw.Ready,
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
					"solvers": http01Solvers(gw),
				},
			},
		},
	}, pulumi.Parent(comp), pulumi.DependsOn([]pulumi.Resource{comp.Release}))
	if err != nil {
		return nil, fmt.Errorf("cert_manager: cluster issuer: %w", err)
	}

	comp.ClusterIssuer = issuer
	args.Cluster.SetClusterIssuer(ClusterIssuerName)

	return comp, nil
}

// values turns on cert-manager's Gateway support, which issues a certificate
// for each HTTPS listener of an annotated Gateway.
func values() pulumi.Map {
	return helm.Values(map[string]any{
		"installCRDs": true,
		"config": map[string]any{
			"gatewayAPI": map[string]any{"enabled": true},
		},
	})
}

func http01Solvers(gw *infra.Gateway) pulumi.Array {
	return pulumi.Array{
		pulumi.Map{
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
		},
	}
}
