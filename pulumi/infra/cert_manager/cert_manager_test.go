package cert_manager

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/internal/pulumitest"
)

func TestNew_requiresAGateway(t *testing.T) {
	t.Parallel()

	for name, c := range map[string]*infra.Cluster{"no gateway": {}, "nil cluster": nil} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var err error
			pulumitest.Run(t, func(ctx *pulumi.Context) error {
				_, err = New(ctx, "cert-manager", &Args{
					Namespace: "infra",
					AcmeEmail: "certs@example.com",
					Cluster:   c,
				})

				return nil
			})
			if err == nil {
				t.Fatal("expected an error without a Gateway to answer challenges through")
			}
		})
	}
}

func TestHTTP01Solvers_answerThroughTheGatewaysHTTPListener(t *testing.T) {
	t.Parallel()

	c := &infra.Cluster{}
	c.SetGateway("gateway", "infra")

	solvers := http01Solvers(c.Gateway)
	if len(solvers) != 1 {
		t.Fatalf("solvers = %v, want the Gateway's", solvers)
	}

	route, _ := solvers[0].(pulumi.Map)["http01"].(pulumi.Map)["gatewayHTTPRoute"].(pulumi.Map)
	refs, _ := route["parentRefs"].(pulumi.Array)
	if len(refs) != 1 {
		t.Fatalf("parentRefs = %v, want the Gateway", route["parentRefs"])
	}
	ref, _ := refs[0].(pulumi.Map)
	if ref["name"] != pulumi.String("gateway") || ref["namespace"] != pulumi.String("infra") ||
		ref["sectionName"] != pulumi.String(infra.GatewayHTTPListener) {
		t.Errorf("parentRef = %v, want the Gateway's HTTP listener", ref)
	}
}
