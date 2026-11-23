package grafana

import "testing"

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
