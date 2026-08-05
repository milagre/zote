package body

import (
	"testing"

	"github.com/milagre/zote/pulumi/k8s/deployment/internal/scaledobject"
	"github.com/milagre/zote/pulumi/util/profile"
)

// TestSeedReplicas pins the bootstrap rule: a workload that autoscales to zero
// is *created* with one replica so its first pod can declare the queue its own
// trigger reads. Every other combination is created at the profile floor.
func TestSeedReplicas(t *testing.T) {
	autoscale := &scaledobject.Spec{Queue: &scaledobject.QueueTrigger{Queue: "jobs"}}

	tests := []struct {
		name      string
		autoscale *scaledobject.Spec
		min       int
		want      int
	}{
		{"scale to zero is seeded with one", autoscale, 0, 1},
		{"nonzero floor is untouched", autoscale, 2, 2},
		{"no autoscaler stays at zero floor", nil, 0, 0},
		{"no autoscaler stays at fixed count", nil, 3, 3},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			args := Args{
				Autoscale: tc.autoscale,
				Profile:   profile.Profile{Num: &profile.IntRange{Min: tc.min, Max: 5}},
			}
			if got := seedReplicas(args); got != tc.want {
				t.Fatalf("seedReplicas() = %d, want %d", got, tc.want)
			}
		})
	}
}
