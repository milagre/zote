// Package mimir installs Grafana Mimir.
package mimir

import (
	"fmt"
	"net/url"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra/grafana_stack/mimir/internal/distributed"
	"github.com/milagre/zote/pulumi/infra/grafana_stack/mimir/internal/monolithic"
	"github.com/milagre/zote/pulumi/infra/objectstorage"
	"github.com/milagre/zote/pulumi/tokens"
)

type Args struct {
	Env       env.Env
	Namespace string

	Config Config

	ObjectStorage objectstorage.ObjectStorage
}

type Mimir struct {
	pulumi.ResourceState

	Gateway    url.URL
	Prometheus url.URL
	Push       url.URL

	// PushURL (and gateway/prometheus) track [Push] etc. but depend on deployed resources,
	// so downstream Helm configs wait until Mimir is registered.
	PushURL       pulumi.StringOutput
	GatewayURL    pulumi.StringOutput
	PrometheusURL pulumi.StringOutput

	Deps []pulumi.Resource // pass to DependsOn
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Mimir, error) {
	if args == nil {
		return nil, fmt.Errorf("mimir: args is required")
	}
	if err := args.validate(); err != nil {
		return nil, fmt.Errorf("mimir: %w", err)
	}

	resourceName := tokens.Qualify(args.Namespace, name)

	comp := &Mimir{}
	if err := ctx.RegisterComponentResource(tokens.Token("infra", "Mimir"), resourceName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering mimir: %w", err)
	}

	switch {
	case args.Config.Monolithic:
		res, err := monolithic.Deploy(ctx, comp, &monolithic.Args{
			Env:           args.Env,
			Namespace:     args.Namespace,
			Name:          name,
			ObjectStorage: args.ObjectStorage,
			Bucket:        args.Config.Bucket,
		})
		if err != nil {
			return nil, fmt.Errorf("mimir: monolithic: %w", err)
		}

		comp.Gateway, comp.Prometheus, comp.Push = res.Gateway, res.Prometheus, res.Push
		comp.Deps = []pulumi.Resource{res.Deployment, res.Service}
		comp.PushURL, comp.GatewayURL, comp.PrometheusURL = urlsAfterID(
			res.Service.ID(), res.Push, res.Gateway, res.Prometheus)

	default:
		res, err := distributed.Deploy(ctx, comp, &distributed.Args{
			Env:           args.Env,
			Namespace:     args.Namespace,
			Name:          name,
			Bucket:        args.Config.Bucket,
			ChartVersion:  args.Config.Version,
			ObjectStorage: args.ObjectStorage,
		})
		if err != nil {
			return nil, fmt.Errorf("mimir: distributed: %w", err)
		}

		comp.Gateway, comp.Prometheus, comp.Push = res.Gateway, res.Prometheus, res.Push
		comp.Deps = []pulumi.Resource{res.Release}
		comp.PushURL, comp.GatewayURL, comp.PrometheusURL = urlsAfterID(
			res.Release.ID(), res.Push, res.Gateway, res.Prometheus)
	}

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"pushUrl":       comp.PushURL,
		"gatewayUrl":    comp.GatewayURL,
		"prometheusUrl": comp.PrometheusURL,
	}); err != nil {
		return nil, fmt.Errorf("mimir: registering outputs: %w", err)
	}

	return comp, nil
}

// urlsAfterID returns StringOutputs tied to the backing resource ID so downstream
// configs inherit the same completion edge as Mimir itself.
func urlsAfterID(
	id pulumi.IDOutput,
	push, gateway, prom url.URL,
) (pushO, gatewayO, promO pulumi.StringOutput) {
	ps := push.String()
	gw := gateway.String()
	pr := prom.String()
	pushO = id.ApplyT(func(pulumi.ID) string { return ps }).(pulumi.StringOutput)
	gatewayO = id.ApplyT(func(pulumi.ID) string { return gw }).(pulumi.StringOutput)
	promO = id.ApplyT(func(pulumi.ID) string { return pr }).(pulumi.StringOutput)

	return pushO, gatewayO, promO
}

func (a *Args) validate() error {
	if a.Namespace == "" {
		return fmt.Errorf("Namespace is required")
	}
	if err := a.Env.Validate(); err != nil {
		return fmt.Errorf("invalid env: %w", err)
	}
	if err := a.Config.Validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if !bucketIn(a.Config.Bucket, a.ObjectStorage.Buckets) {
		return fmt.Errorf("ObjectStorage does not contain bucket %q", a.Config.Bucket)
	}

	return nil
}

func bucketIn(bucket string, buckets []string) bool {
	for _, b := range buckets {
		if b == bucket {
			return true
		}
	}

	return false
}
