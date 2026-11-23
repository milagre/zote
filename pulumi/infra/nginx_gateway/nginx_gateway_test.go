package nginx_gateway_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/infra/nginx_gateway"
	"github.com/milagre/zote/pulumi/internal/pulumitest"
	"github.com/milagre/zote/pulumi/util/profile"
)

const (
	gatewayType = "kubernetes:gateway.networking.k8s.io/v1:Gateway"
	routeType   = "kubernetes:gateway.networking.k8s.io/v1:HTTPRoute"
)

// run installs the controller, registers hosts the way routes do, then
// creates the Gateway.
func run(t *testing.T, typ string, issuer bool, hosts ...string) ([]pulumitest.Resource, error) {
	t.Helper()

	e, err := env.New("zote", typ, "prod", "us-e1", "example.com", "z")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	c := &infra.Cluster{}
	if issuer {
		c.SetClusterIssuer("letsencrypt-http01")
	}

	var createErr error
	rs := pulumitest.Run(t, func(ctx *pulumi.Context) error {
		gw, err := nginx_gateway.New(ctx, "nginx-gateway", &nginx_gateway.Args{
			Namespace: "infra",
			Env:       e,
			Config: nginx_gateway.Config{
				Profile: profile.Raw{
					CPU: profile.RawRange{Min: "100m", Max: "200m"},
					Mem: profile.RawRange{Min: "64M", Max: "128M"},
				},
				ExternalCRDs: true,
			},
			Cluster: c,
		})
		if err != nil {
			return err
		}

		c.Gateway.AddHostnames(hosts...)
		createErr = gw.CreateGateway(ctx)

		return nil
	})

	return rs, createErr
}

type listener struct {
	name, hostname, protocol, from, secret string
}

func listeners(t *testing.T, rs []pulumitest.Resource) (map[string]any, []listener) {
	t.Helper()

	gateways := pulumitest.OfType(rs, gatewayType)
	if len(gateways) != 1 {
		t.Fatalf("gateways = %d, want 1", len(gateways))
	}

	var out []listener
	list, _ := pulumitest.Lookup(gateways[0].Inputs, "spec", "listeners").([]any)
	for _, l := range list {
		var secret string
		if refs, _ := pulumitest.Lookup(l, "tls", "certificateRefs").([]any); len(refs) == 1 {
			secret, _ = pulumitest.Lookup(refs[0], "name").(string)
		}
		name, _ := pulumitest.Lookup(l, "name").(string)
		hostname, _ := pulumitest.Lookup(l, "hostname").(string)
		protocol, _ := pulumitest.Lookup(l, "protocol").(string)
		from, _ := pulumitest.Lookup(l, "allowedRoutes", "namespaces", "from").(string)
		out = append(out, listener{name, hostname, protocol, from, secret})
	}

	ann, _ := pulumitest.Lookup(gateways[0].Inputs, "metadata", "annotations").(map[string]any)

	return ann, out
}

func TestCreateGateway_tlsListenerPerHostname(t *testing.T) {
	t.Parallel()

	rs, err := run(t, "remote", true, "a.example.com", "b.example.com")
	if err != nil {
		t.Fatalf("CreateGateway: %v", err)
	}

	ann, ls := listeners(t, rs)
	if ann["cert-manager.io/cluster-issuer"] != "letsencrypt-http01" {
		t.Errorf("annotations = %v, want the cluster issuer", ann)
	}
	if len(ls) != 3 {
		t.Fatalf("listeners = %+v, want http plus one per hostname", ls)
	}
	if ls[0].name != infra.GatewayHTTPListener || ls[0].protocol != "HTTP" || ls[0].from != "Same" {
		t.Errorf("first listener = %+v, want plain HTTP for this namespace's routes", ls[0])
	}

	secrets := map[string]bool{}
	for i, host := range []string{"a.example.com", "b.example.com"} {
		l := ls[i+1]
		if l.name != infra.GatewayListenerName(host) || l.hostname != host || l.protocol != "HTTPS" || l.from != "All" {
			t.Errorf("listener %d = %+v, want HTTPS for %s open to all namespaces", i+1, l, host)
		}
		if l.secret == "" || secrets[l.secret] {
			t.Errorf("listener %s certificate secret %q is missing or shared", l.name, l.secret)
		}
		secrets[l.secret] = true
	}

	routes := pulumitest.OfType(rs, routeType)
	if len(routes) != 1 {
		t.Fatalf("routes = %d, want the HTTPS redirect", len(routes))
	}
	rules, _ := pulumitest.Lookup(routes[0].Inputs, "spec", "rules").([]any)
	filters, _ := pulumitest.Lookup(rules[0], "filters").([]any)
	if len(filters) != 1 || pulumitest.Lookup(filters[0], "requestRedirect", "scheme") != "https" {
		t.Errorf("redirect filters = %v, want a redirect to https", filters)
	}
}

func TestCreateGateway_plainHTTP(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		typ    string
		issuer bool
	}{
		"local":                 {"local", false},
		"remote without issuer": {"remote", false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rs, err := run(t, tc.typ, tc.issuer, "a.example.com")
			if err != nil {
				t.Fatalf("CreateGateway: %v", err)
			}

			ann, ls := listeners(t, rs)
			if _, ok := ann["cert-manager.io/cluster-issuer"]; ok {
				t.Errorf("annotations = %v, want no issuer", ann)
			}
			want := []listener{{name: infra.GatewayHTTPListener, protocol: "HTTP", from: "All"}}
			if !slices.Equal(ls, want) {
				t.Errorf("listeners = %+v, want %+v", ls, want)
			}
			if routes := pulumitest.OfType(rs, routeType); len(routes) != 0 {
				t.Errorf("routes = %d, want no redirect", len(routes))
			}
		})
	}
}

func TestCreateGateway_rejectsMoreListenersThanAGatewayHolds(t *testing.T) {
	t.Parallel()

	hosts := make([]string, 64)
	for i := range hosts {
		hosts[i] = fmt.Sprintf("h%d.example.com", i)
	}

	if _, err := run(t, "remote", true, hosts...); err == nil {
		t.Fatal("expected an error for 65 listeners")
	}
}
