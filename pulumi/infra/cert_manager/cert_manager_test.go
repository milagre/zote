package cert_manager

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/infra"
)

func gatewayCluster() *infra.Cluster {
	c := &infra.Cluster{}
	c.SetGateway("public", "infra")

	return c
}

func TestValues_gatewaySupportFollowsTheCluster(t *testing.T) {
	t.Parallel()

	on, ok := values(&Args{Cluster: gatewayCluster()})["config"].(pulumi.Map)
	if !ok {
		t.Fatal("config: not set with a Gateway on the cluster")
	}
	if gw, _ := on["gatewayAPI"].(pulumi.Map); gw["enabled"] != pulumi.Bool(true) {
		t.Errorf("config.gatewayAPI = %v, want enabled", on["gatewayAPI"])
	}

	for name, c := range map[string]*infra.Cluster{"no gateway": {}, "nil cluster": nil} {
		if _, ok := values(&Args{Cluster: c})["config"]; ok {
			t.Errorf("%s: config set, want the chart default", name)
		}
	}
}

func TestHTTP01Solvers_gatewaySolverFollowsTheIngressClasses(t *testing.T) {
	t.Parallel()

	solvers := http01Solvers(&Args{
		IngressClasses: []string{"nginx"},
		Cluster:        gatewayCluster(),
	})
	if len(solvers) != 2 {
		t.Fatalf("solvers = %v, want the ingress class then the Gateway", solvers)
	}

	ingress, _ := solvers[0].(pulumi.Map)["http01"].(pulumi.Map)
	if _, ok := ingress["ingress"]; !ok {
		t.Errorf("first solver = %v, want the ingress class", solvers[0])
	}

	route, _ := solvers[1].(pulumi.Map)["http01"].(pulumi.Map)["gatewayHTTPRoute"].(pulumi.Map)
	refs, _ := route["parentRefs"].(pulumi.Array)
	if len(refs) != 1 {
		t.Fatalf("parentRefs = %v, want the Gateway", route["parentRefs"])
	}
	ref, _ := refs[0].(pulumi.Map)
	if ref["name"] != pulumi.String("public") || ref["namespace"] != pulumi.String("infra") ||
		ref["sectionName"] != pulumi.String(infra.GatewayHTTPListener) {
		t.Errorf("parentRef = %v, want the Gateway's HTTP listener", ref)
	}
}

func TestHTTP01Solvers_gatewaySolverNeedsAGateway(t *testing.T) {
	t.Parallel()

	if solvers := http01Solvers(&Args{Cluster: &infra.Cluster{}}); len(solvers) != 0 {
		t.Errorf("solvers = %v, want none without a Gateway", solvers)
	}
}
