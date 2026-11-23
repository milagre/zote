package nginx_gateway

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/util/profile"
)

func chartValues(t *testing.T, typ string, num *profile.IntRange) pulumi.Map {
	t.Helper()

	e, err := env.New("zote", typ, "prod", "us-e1", "example.com", "z")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	p, err := profile.New(profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "200m"},
		Mem: profile.RawRange{Min: "64M", Max: "128M"},
		Num: num,
	})
	if err != nil {
		t.Fatalf("profile.New: %v", err)
	}

	return values(&Args{Env: e}, p, "public")
}

func lookup(t *testing.T, m pulumi.Map, keys ...string) pulumi.Input {
	t.Helper()

	var v pulumi.Input = m
	for _, k := range keys {
		mm, ok := v.(pulumi.Map)
		if !ok {
			t.Fatalf("%v: not a pulumi.Map at %q", keys, k)
		}
		v = mm[k]
	}

	return v
}

func TestValues_serviceType(t *testing.T) {
	t.Parallel()

	for typ, want := range map[string]string{"remote": "LoadBalancer", "local": "NodePort"} {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()

			if got := lookup(t, chartValues(t, typ, nil), "nginx", "service", "type"); got != pulumi.String(want) {
				t.Errorf("service type = %v, want %s", got, want)
			}
		})
	}
}

func TestValues_telemetryOff(t *testing.T) {
	t.Parallel()

	if got := lookup(t, chartValues(t, "remote", nil), "nginxGateway", "productTelemetry", "enable"); got != pulumi.Bool(false) {
		t.Errorf("productTelemetry.enable = %v, want false", got)
	}
}

func TestValues_replicasFollowProfile(t *testing.T) {
	t.Parallel()

	v := chartValues(t, "remote", &profile.IntRange{Min: 2, Max: 5})
	if got := lookup(t, v, "nginx", "autoscaling", "minReplicas"); got != pulumi.Int(2) {
		t.Errorf("minReplicas = %v, want 2", got)
	}
	if got := lookup(t, v, "nginx", "autoscaling", "maxReplicas"); got != pulumi.Int(5) {
		t.Errorf("maxReplicas = %v, want 5", got)
	}
}

func TestValues_dataPlaneIsScraped(t *testing.T) {
	t.Parallel()

	patches, ok := lookup(t, chartValues(t, "remote", nil), "nginx", "patches").(pulumi.Array)
	if !ok || len(patches) != 1 {
		t.Fatalf("nginx.patches = %v, want one patch", patches)
	}

	patch, _ := patches[0].(pulumi.Map)
	if patch["type"] != pulumi.String("StrategicMerge") {
		t.Errorf("patch type = %v, want StrategicMerge", patch["type"])
	}

	ann, _ := lookup(t, patch, "value", "spec", "template", "metadata", "annotations").(pulumi.StringMap)
	if ann["prometheus.io/scrape"] != pulumi.String("true") || ann["prometheus.io/port"] != pulumi.String("9113") {
		t.Errorf("pod annotations = %v, want scraping on the agent's metrics port", ann)
	}
}
