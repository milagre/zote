// Package cloudflare_tunnel installs strrl.dev cloudflare-tunnel-ingress-controller.
package cloudflare_tunnel

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/tokens"
)

var spec = helm.ChartSpec{
	TypeToken:  tokens.Token("infra", "CloudflareTunnel"),
	Chart:      "cloudflare-tunnel-ingress-controller",
	Repository: "https://helm.strrl.dev/",
}

const IngressClassName = "cloudflare-tunnel" // chart-fixed IngressClass

type Args struct {
	Namespace  string
	AccountID  string
	APIToken   string
	TunnelName string

	Config Config
}

type CloudflareTunnel struct {
	helm.ChartComponent

	IngressClassName pulumi.StringOutput
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*CloudflareTunnel, error) {
	if args == nil {
		return nil, fmt.Errorf("cloudflare_tunnel: args is required")
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("cloudflare_tunnel: Namespace is required")
	}
	if args.AccountID == "" || args.APIToken == "" || args.TunnelName == "" {
		return nil, fmt.Errorf("cloudflare_tunnel: AccountID, APIToken, TunnelName are required")
	}
	if err := args.Config.Validate(); err != nil {
		return nil, fmt.Errorf("cloudflare_tunnel: config: %w", err)
	}

	comp := &CloudflareTunnel{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   helm.OptionalChartVersion(args.Config.Version),
		Values: pulumi.Map{
			"cloudflare": pulumi.Map{
				"apiToken":   pulumi.String(args.APIToken),
				"accountId":  pulumi.String(args.AccountID),
				"tunnelName": pulumi.String(args.TunnelName),
			},
		},
	}, &comp.ChartComponent, opts...); err != nil {
		return nil, err
	}

	comp.IngressClassName = pulumi.String(IngressClassName).ToStringOutput()

	return comp, nil
}
