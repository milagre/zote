package httproute_test

import (
	"slices"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/internal/httproute"
	"github.com/milagre/zote/pulumi/internal/pulumitest"
)

const (
	routeType       = "kubernetes:gateway.networking.k8s.io/v1:HTTPRoute"
	listenerSetType = "kubernetes:gateway.networking.k8s.io/v1:ListenerSet"
)

func newEnv(t *testing.T, typ string) env.Env {
	t.Helper()

	e, err := env.New("zote", typ, "prod", "us-e1", "example.com", "z")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	return e
}

func gatewayCluster(issuer bool) *infra.Cluster {
	c := &infra.Cluster{}
	c.SetGateway("gateway", "infra")
	if issuer {
		c.SetClusterIssuer("letsencrypt-http01")
	}

	return c
}

func register(t *testing.T, e env.Env, c *infra.Cluster, hosts ...string) []pulumitest.Resource {
	t.Helper()

	return pulumitest.Run(t, func(ctx *pulumi.Context) error {
		_, err := httproute.Register(ctx, "apps/my-api", httproute.Args{
			Env:       e,
			Namespace: "apps",
			Name:      "my-api",
			Hostnames: hosts,
			Service:   pulumi.String("my-api"),
			Port:      80,
			Cluster:   c,
		})

		return err
	})
}

// parents returns the kind, name and section of each of route's parentRefs.
func parents(route pulumitest.Resource) [][3]string {
	refs, _ := pulumitest.Lookup(route.Inputs, "spec", "parentRefs").([]any)
	out := make([][3]string, 0, len(refs))
	for _, r := range refs {
		kind, _ := pulumitest.Lookup(r, "kind").(string)
		name, _ := pulumitest.Lookup(r, "name").(string)
		section, _ := pulumitest.Lookup(r, "sectionName").(string)
		out = append(out, [3]string{kind, name, section})
	}

	return out
}

func setsResponseHeader(route pulumitest.Resource, name string) bool {
	rules, _ := pulumitest.Lookup(route.Inputs, "spec", "rules").([]any)
	for _, rule := range rules {
		filters, _ := pulumitest.Lookup(rule, "filters").([]any)
		for _, f := range filters {
			set, _ := pulumitest.Lookup(f, "responseHeaderModifier", "set").([]any)
			for _, h := range set {
				if pulumitest.Lookup(h, "name") == name {
					return true
				}
			}
		}
	}

	return false
}

func TestRegister_tlsServesEachHostnameOnItsOwnListener(t *testing.T) {
	t.Parallel()

	hosts := []string{"my-api.apps.example.com", "api.example.com"}
	rs := register(t, newEnv(t, "remote"), gatewayCluster(true), hosts...)

	sets := pulumitest.OfType(rs, listenerSetType)
	if len(sets) != 1 {
		t.Fatalf("listener sets = %d, want 1", len(sets))
	}
	set := sets[0]
	if pulumitest.Lookup(set.Inputs, "spec", "parentRef", "name") != "gateway" ||
		pulumitest.Lookup(set.Inputs, "spec", "parentRef", "namespace") != "infra" {
		t.Errorf("parentRef = %v, want the cluster's Gateway", pulumitest.Lookup(set.Inputs, "spec", "parentRef"))
	}
	if pulumitest.Lookup(set.Inputs, "metadata", "annotations", "cert-manager.io/cluster-issuer") != "letsencrypt-http01" {
		t.Errorf("annotations = %v, want the cluster issuer", pulumitest.Lookup(set.Inputs, "metadata", "annotations"))
	}

	setName, _ := pulumitest.Lookup(set.Inputs, "metadata", "name").(string)
	listeners, _ := pulumitest.Lookup(set.Inputs, "spec", "listeners").([]any)
	if len(listeners) != len(hosts) {
		t.Fatalf("listeners = %v, want one per hostname", listeners)
	}
	secrets := map[any]bool{}
	for i, host := range hosts {
		l := listeners[i]
		if pulumitest.Lookup(l, "name") != infra.GatewayListenerName(host) || pulumitest.Lookup(l, "hostname") != host ||
			pulumitest.Lookup(l, "protocol") != "HTTPS" {
			t.Errorf("listener %d = %v, want HTTPS for %s", i, l, host)
		}
		refs, _ := pulumitest.Lookup(l, "tls", "certificateRefs").([]any)
		if len(refs) != 1 || secrets[pulumitest.Lookup(refs[0], "name")] {
			t.Errorf("listener %s certificateRefs = %v, want a secret of its own", host, refs)
		} else {
			secrets[pulumitest.Lookup(refs[0], "name")] = true
		}
	}

	routes := pulumitest.OfType(rs, routeType)
	if len(routes) != 1 {
		t.Fatalf("routes = %d, want 1", len(routes))
	}
	want := [][3]string{
		{"ListenerSet", setName, infra.GatewayListenerName(hosts[0])},
		{"ListenerSet", setName, infra.GatewayListenerName(hosts[1])},
	}
	if got := parents(routes[0]); !slices.Equal(got, want) {
		t.Errorf("parentRefs = %v, want %v", got, want)
	}
	if !setsResponseHeader(routes[0], "Strict-Transport-Security") {
		t.Error("TLS route sets no Strict-Transport-Security header")
	}
}

func TestRegister_localAttachesToHTTPListener(t *testing.T) {
	t.Parallel()

	rs := register(t, newEnv(t, "local"), gatewayCluster(false), "my-api.apps.example.com")
	if sets := pulumitest.OfType(rs, listenerSetType); len(sets) != 0 {
		t.Errorf("listener sets = %d, want none", len(sets))
	}

	routes := pulumitest.OfType(rs, routeType)
	if len(routes) != 1 {
		t.Fatalf("routes = %d, want 1", len(routes))
	}
	if got := parents(routes[0]); !slices.Equal(got, [][3]string{{"Gateway", "gateway", infra.GatewayHTTPListener}}) {
		t.Errorf("parentRefs = %v, want the Gateway's HTTP listener", got)
	}
	if setsResponseHeader(routes[0], "Strict-Transport-Security") {
		t.Error("plain HTTP route sets Strict-Transport-Security")
	}
}

func TestRegister_skipsWithoutAWayToServe(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		typ     string
		cluster *infra.Cluster
	}{
		"no gateway":            {"remote", &infra.Cluster{}},
		"nil cluster":           {"local", nil},
		"tls without an issuer": {"remote", gatewayCluster(false)},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if rs := register(t, newEnv(t, tc.typ), tc.cluster, "my-api.apps.example.com"); len(rs) != 0 {
				t.Errorf("resources = %v, want none", rs)
			}
		})
	}
}
