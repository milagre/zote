// Package cloudflare_tunnel installs the strrl.dev cloudflare-tunnel
// ingress-controller Helm chart. The tunnel's three identifying
// inputs — account id, api token, tunnel name — are the only thing
// that varies between deployments.
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

// IngressClassName is the name of the IngressClass resource the
// upstream chart registers and that Ingress objects need to reference
// (via spec.ingressClassName or kubernetes.io/ingress.class) to route
// through the tunnel. The chart hardcodes this name, so it's exposed
// as a constant rather than plumbed through as a chart value.
const IngressClassName = "cloudflare-tunnel"

// Args are the caller-supplied inputs.
type Args struct {
	// Namespace is the target namespace. Required.
	Namespace string

	// AccountID is the Cloudflare account the tunnel lives under.
	// Required.
	AccountID string

	// APIToken authorises the controller to manage the tunnel.
	// Required. The caller is responsible for sourcing it out of
	// encrypted stack config; once in hand it is a plain string
	// here.
	APIToken string

	// TunnelName is the Cloudflare tunnel to attach to. The
	// controller will create it if absent. Required.
	TunnelName string

	// Version overrides the chart's pinned version. Optional.
	Version *string
}

// CloudflareTunnel is the chart install as a ComponentResource.
type CloudflareTunnel struct {
	helm.ChartComponent

	// IngressClassName is the name of the IngressClassName Ingress resources
	// must reference to route through the tunnel. Surfaced as an
	// Output so consumers can thread it into ingress specs without
	// rediscovering the chart's contract.
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

	comp := &CloudflareTunnel{}
	if err := helm.RegisterChart(ctx, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   args.Version,
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
