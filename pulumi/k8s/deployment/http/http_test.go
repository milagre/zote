package http_test

import (
	"slices"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/internal/pulumitest"
	"github.com/milagre/zote/pulumi/k8s/deployment/http"
	"github.com/milagre/zote/pulumi/util/profile"
)

const (
	ingressType = "kubernetes:networking.k8s.io/v1:Ingress"
	routeType   = "kubernetes:gateway.networking.k8s.io/v1:HTTPRoute"
)

func run(t *testing.T, typ string, c *infra.Cluster) []pulumitest.Resource {
	t.Helper()

	e, err := env.New("zote", typ, "prod", "us-e1", "example.com", "z")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	return pulumitest.Run(t, func(ctx *pulumi.Context) error {
		_, err := http.New(ctx, "my-api", &http.Args{
			Env:       e,
			Namespace: "apps",
			Name:      "my-api",
			Image:     "my-api",
			Tag:       "1.0.0",
			Profile:   profile.Profile{Num: &profile.IntRange{Min: 1, Max: 1}},
			Setup:     http.Options{Port: 8080, Health: "/health"},
			Internal: http.Internal{
				PublicHostnames: []string{"my-api.apps.example.com"},
				VeneerHostnames: []string{"api.example.com"},
			},
			Cluster: c,
		})

		return err
	})
}

func cluster() *infra.Cluster {
	c := &infra.Cluster{}
	c.SetTunnelIngressClass("cloudflare-tunnel")
	c.SetClusterIssuer("letsencrypt-http01")
	c.SetGateway("gateway", "infra")

	return c
}

func backendServices(paths any) []string {
	var out []string
	list, _ := paths.([]any)
	for _, p := range list {
		if s, ok := pulumitest.Lookup(p, "backend", "service", "name").(string); ok {
			out = append(out, s)
		}
	}

	return out
}

func TestNew_localPublicWorkloadKeepsItsTunnelIngress(t *testing.T) {
	t.Parallel()

	var tunnel *pulumitest.Resource
	for _, r := range pulumitest.OfType(run(t, "local", cluster()), ingressType) {
		if pulumitest.Lookup(r.Inputs, "spec", "ingressClassName") == "cloudflare-tunnel" {
			tunnel = &r
		}
	}
	if tunnel == nil {
		t.Fatal("no cloudflare-tunnel Ingress")
	}

	rules, _ := pulumitest.Lookup(tunnel.Inputs, "spec", "rules").([]any)
	if len(rules) != 1 || pulumitest.Lookup(rules[0], "host") != "my-api.apps.example.com" {
		t.Fatalf("tunnel rules = %v, want the public hostname only", rules)
	}
	if got := backendServices(pulumitest.Lookup(rules[0], "http", "paths")); !slices.Equal(got, []string{"my-api"}) {
		t.Errorf("tunnel backends = %v, want the workload's Service", got)
	}
}

func TestNew_publicWorkloadIsRoutedThroughTheGateway(t *testing.T) {
	t.Parallel()

	for _, typ := range []string{"remote", "local"} {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()

			routes := pulumitest.OfType(run(t, typ, cluster()), routeType)
			if len(routes) != 1 {
				t.Fatalf("routes = %d, want 1", len(routes))
			}

			hosts := pulumitest.Lookup(routes[0].Inputs, "spec", "hostnames")
			want := []any{"my-api.apps.example.com", "api.example.com"}
			if got, _ := hosts.([]any); !slices.Equal(got, want) {
				t.Errorf("hostnames = %v, want %v", hosts, want)
			}

			rules, _ := pulumitest.Lookup(routes[0].Inputs, "spec", "rules").([]any)
			backends, _ := pulumitest.Lookup(rules[0], "backendRefs").([]any)
			if len(backends) != 1 || pulumitest.Lookup(backends[0], "name") != "my-api" {
				t.Errorf("backendRefs = %v, want the workload's Service", backends)
			}
		})
	}
}
