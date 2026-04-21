// Package cert_manager installs the upstream cert-manager Helm chart
// together with a letsencrypt-http01 ClusterIssuer that every ingress
// in the cluster delegates TLS issuance to.
package cert_manager

import (
	"fmt"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/tokens"
)

// ClusterIssuerName is the name of the ClusterIssuer this component
// creates. Exported so ingress annotations can reference it by name.
const ClusterIssuerName = "letsencrypt-http01"

var spec = helm.ChartSpec{
	TypeToken:      tokens.Token("infra", "CertManager"),
	Chart:          "cert-manager",
	Repository:     "https://charts.jetstack.io",
	DefaultVersion: "1.13.1",
}

// Args are the caller-supplied inputs. The ClusterIssuer's
// contact email is the only value that varies between deployments.
type Args struct {
	// Namespace is the target namespace. Required.
	Namespace string

	// AcmeEmail is the contact address Let's Encrypt uses for
	// expiring-cert notifications. Required.
	AcmeEmail string

	// Version overrides DefaultVersion. Optional.
	Version *string
}

// CertManager wraps the chart and the ClusterIssuer as a single
// ComponentResource so callers can express pulumi.DependsOn against
// the whole install, not just one of its pieces.
type CertManager struct {
	helm.ChartComponent

	// ClusterIssuer is the cert-manager.io/v1 ClusterIssuer this
	// component creates, exposed so callers can attach explicit
	// ordering edges if needed.
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

	comp := &CertManager{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   args.Version,
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
					"solvers": pulumi.Array{
						pulumi.Map{
							"http01": pulumi.Map{
								"ingress": pulumi.Map{
									"class": pulumi.String("nginx"),
								},
							},
						},
					},
				},
			},
		},
	}, pulumi.Parent(comp), pulumi.DependsOn([]pulumi.Resource{comp.Release}))
	if err != nil {
		return nil, fmt.Errorf("cert_manager: cluster issuer: %w", err)
	}

	comp.ClusterIssuer = issuer

	return comp, nil
}
