package grafana

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/internal/httproute"
	"github.com/milagre/zote/pulumi/util/tokens"
)

// registerRoute serves Grafana's public hostnames through the cluster's
// Gateway. It returns nil when there is nothing to serve them with.
func registerRoute(ctx *pulumi.Context, name string, args *Args, g *Grafana, parent pulumi.Resource) (pulumi.Resource, error) {
	if len(args.PublicDomains) == 0 {
		return nil, nil
	}

	route, err := httproute.Register(ctx, tokens.Qualify(args.Namespace, name), httproute.Args{
		Env:       args.Env,
		Namespace: args.Namespace,
		Name:      name,
		Hostnames: publicHosts(name, args.Namespace, args.PublicDomains),
		Service:   pulumi.String(name),
		Port:      80,
		Cluster:   args.Cluster,
	}, pulumi.Parent(parent), pulumi.DependsOn(publicDeps(args, g)))
	if err != nil {
		return nil, fmt.Errorf("route: %w", err)
	}
	if route == nil {
		return nil, nil
	}

	return route, nil
}

// publicDeps are what anything serving Grafana publicly waits on.
func publicDeps(args *Args, g *Grafana) []pulumi.Resource {
	deps := append([]pulumi.Resource{}, args.IngressDeps...)
	deps = append(deps, g.Helm.Release)

	return deps
}

// publicHosts returns the fully qualified public hostnames (<name>.<namespace>.<domain>)
// Grafana is served at, one per public domain. The first is the primary.
func publicHosts(name, namespace string, publicDomains []string) []string {
	out := make([]string, 0, len(publicDomains))
	for _, d := range publicDomains {
		out = append(out, fmt.Sprintf("%s.%s.%s", name, namespace, d))
	}

	return out
}
