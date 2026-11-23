package grafana

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
)

func TestPublicEntry_isWhatReachesTheClusterFromOutside(t *testing.T) {
	t.Parallel()

	route, tunnel := &pulumi.ResourceState{}, &pulumi.ResourceState{}

	for name, tc := range map[string]struct {
		typ           string
		route, tunnel pulumi.Resource
		want          pulumi.Resource
	}{
		"remote route":       {"remote", route, nil, route},
		"remote without one": {"remote", nil, nil, nil},
		"local tunnel":       {"local", route, tunnel, tunnel},
		"local route only":   {"local", route, nil, nil},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e, err := env.New("zote", tc.typ, "prod", "us-e1", "example.com", "z")
			if err != nil {
				t.Fatalf("env.New: %v", err)
			}

			if got := publicEntry(e, tc.route, tc.tunnel); got != tc.want {
				t.Errorf("publicEntry = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPublicHosts(t *testing.T) {
	t.Parallel()

	got := publicHosts("grafana", "infra", []string{"example.test", "example.com"})
	want := []string{
		"grafana.infra.example.test",
		"grafana.infra.example.com",
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
