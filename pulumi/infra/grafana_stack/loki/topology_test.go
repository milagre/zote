package loki

import (
	"testing"

	"github.com/milagre/zote/pulumi/env"
)

func TestHelmTopology_localIsSingleBinary(t *testing.T) {
	t.Parallel()

	e, err := env.New("local", "dev", "x", "/tmp", "Z")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	mode, w, r, b, s := helmTopology(e)
	if mode != "SingleBinary" || w != 0 || r != 0 || b != 0 || s != 1 {
		t.Fatalf("got mode=%q write=%d read=%d backend=%d single=%d", mode, w, r, b, s)
	}
}

func TestHelmTopology_remoteIsSimpleScalable(t *testing.T) {
	t.Parallel()

	e, err := env.New("remote", "prod", "x", "/tmp", "Z")
	if err != nil {
		t.Fatalf("env.New: %v", err)
	}

	mode, w, r, b, s := helmTopology(e)
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
