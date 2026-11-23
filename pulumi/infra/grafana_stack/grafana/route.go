package grafana

import (
	"fmt"
	"net/url"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
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

// publicEntry is what carries callers outside the cluster to Grafana's public
// hostnames: the Gateway route, or locally the tunnel Ingress, since a local
// Gateway is not reachable at those hostnames.
func publicEntry(e env.Env, route, tunnel pulumi.Resource) pulumi.Resource {
	if e.IsLocal() {
		return tunnel
	}

	return route
}

// publicDeps are what anything serving Grafana publicly waits on.
func publicDeps(args *Args, g *Grafana) []pulumi.Resource {
	deps := append([]pulumi.Resource{}, args.PublicDeps...)
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

// publicURL is the HTTPS endpoint for host; the Gateway and the tunnel both
// terminate TLS.
func publicURL(host string) url.URL {
	return url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/",
	}
}
