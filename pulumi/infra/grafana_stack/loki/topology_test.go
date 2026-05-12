package loki

import "testing"

func TestHelmTopology_monolithicIsSingleBinary(t *testing.T) {
	t.Parallel()

	mode, w, r, b, s := helmTopology(true)
	if mode != "SingleBinary" || w != 0 || r != 0 || b != 0 || s != 1 {
		t.Fatalf("got mode=%q write=%d read=%d backend=%d single=%d", mode, w, r, b, s)
	}
}

func TestHelmTopology_notMonolithicIsSimpleScalable(t *testing.T) {
	t.Parallel()

	mode, w, r, b, s := helmTopology(false)
	if mode != "SimpleScalable" || w != 2 || r != 2 || b != 2 || s != 0 {
		t.Fatalf("got mode=%q write=%d read=%d backend=%d single=%d", mode, w, r, b, s)
	}
}

func TestReplicationFactor(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		writeReplicas int
		want          int
	}{
		{0, 1},
		{1, 1},
		{2, 2},
	} {
		if got := replicationFactor(tt.writeReplicas); got != tt.want {
			t.Fatalf("replicationFactor(%d)=%d want %d", tt.writeReplicas, got, tt.want)
		}
	}
}
