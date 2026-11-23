package nginx_gateway_test

import (
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

func run(t *testing.T, typ string) (pulumitest.Resource, []pulumitest.Resource) {
	t.Helper()

	e, err := env.New("zote", typ, "prod", "us-e1", "example.com", "z")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	c := &infra.Cluster{}
	rs := pulumitest.Run(t, func(ctx *pulumi.Context) error {
		_, err := nginx_gateway.New(ctx, "nginx-gateway", &nginx_gateway.Args{
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

		return err
	})

	if c.Gateway == nil || c.Gateway.Name != nginx_gateway.DefaultGatewayName || c.Gateway.Namespace != "infra" {
		t.Fatalf("cluster Gateway = %+v, want the default name in infra", c.Gateway)
	}

	gateways := pulumitest.OfType(rs, gatewayType)
	if len(gateways) != 1 {
		t.Fatalf("gateways = %d, want 1", len(gateways))
	}

	return gateways[0], pulumitest.OfType(rs, routeType)
}

// The only listener is plain HTTP, open to every namespace: locally routes
// attach to it, and elsewhere ACME solvers do, from each certificate's own
// namespace.
func assertOnlyHTTPListener(t *testing.T, gw pulumitest.Resource) {
	t.Helper()

	listeners, _ := pulumitest.Lookup(gw.Inputs, "spec", "listeners").([]any)
	if len(listeners) != 1 {
		t.Fatalf("listeners = %v, want one", listeners)
	}

	l := listeners[0]
	if pulumitest.Lookup(l, "name") != infra.GatewayHTTPListener || pulumitest.Lookup(l, "protocol") != "HTTP" ||
		pulumitest.Lookup(l, "allowedRoutes", "namespaces", "from") != "All" {
		t.Errorf("listener = %v, want plain HTTP open to all namespaces", l)
	}
}

func TestNew_remoteGatewayTakesListenerSetsAndRedirects(t *testing.T) {
	t.Parallel()

	gw, routes := run(t, "remote")
	assertOnlyHTTPListener(t, gw)

	if got := pulumitest.Lookup(gw.Inputs, "spec", "allowedListeners", "namespaces", "from"); got != "All" {
		t.Errorf("allowedListeners from = %v, want All", got)
	}

	if len(routes) != 1 {
		t.Fatalf("routes = %d, want the HTTPS redirect", len(routes))
	}
	rules, _ := pulumitest.Lookup(routes[0].Inputs, "spec", "rules").([]any)
	filters, _ := pulumitest.Lookup(rules[0], "filters").([]any)
	if len(filters) != 1 || pulumitest.Lookup(filters[0], "requestRedirect", "scheme") != "https" {
		t.Errorf("redirect filters = %v, want a redirect to https", filters)
	}
}

func TestNew_localGatewayServesPlainHTTP(t *testing.T) {
	t.Parallel()

	gw, routes := run(t, "local")
	assertOnlyHTTPListener(t, gw)

	if got := pulumitest.Lookup(gw.Inputs, "spec", "allowedListeners"); got != nil {
		t.Errorf("allowedListeners = %v, want none", got)
	}
	if len(routes) != 0 {
		t.Errorf("routes = %d, want no redirect", len(routes))
	}
}
