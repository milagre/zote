// Package local renders Loki helm values for SingleBinary mode backed by a PV.
package local

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/profile"
)

const defaultSize = "10Gi"

type Args struct {
	Profile profile.Profile
	Spec    Spec
}

func (a *Args) Setup(_ *pulumi.Context, _ pulumi.Resource) (pulumi.Map, []pulumi.Resource, error) {
	return valuesFor(a), nil, nil
}

// valuesFor renders SingleBinary with Loki's chart "filesystem" storage type
// (PV-backed). write/read/backend replicas are zeroed so the chart doesn't
// stand up StatefulSets that would crash without object storage; caches and
// helper pods are off because SingleBinary doesn't need them.
func valuesFor(a *Args) pulumi.Map {
	size := defaultSize
	if a.Spec.Size != nil {
		size = *a.Spec.Size
	}

	return helm.Values(map[string]any{
		"deploymentMode": "SingleBinary",
		"loki": map[string]any{
			"commonConfig": map[string]any{
				"replication_factor": 1,
			},
			"storage": map[string]any{
				"type": "filesystem",
			},
			"schemaConfig": map[string]any{
				"configs": []any{
					map[string]any{
						"from":         "2024-04-01",
						"store":        "tsdb",
						"object_store": "filesystem",
						"schema":       "v13",
						"index": map[string]any{
							"prefix": "index_",
							"period": "24h",
						},
					},
				},
			},
		},
		"singleBinary": map[string]any{
			"replicas": 1,
			"persistence": map[string]any{
				"enabled": true,
				"size":    size,
			},
			"resources": resourcesFor(a.Profile),
		},
		"write":        map[string]any{"replicas": 0},
		"read":         map[string]any{"replicas": 0},
		"backend":      map[string]any{"replicas": 0},
		"chunksCache":  map[string]any{"enabled": false},
		"resultsCache": map[string]any{"enabled": false},
		"test":         map[string]any{"enabled": false},
		"lokiCanary":   map[string]any{"enabled": false},
		"monitoring": map[string]any{
			"selfMonitoring": map[string]any{"enabled": false},
		},
	})
}

func resourcesFor(p profile.Profile) map[string]any {
	return map[string]any{
		"requests": map[string]any{
			"cpu":    p.MinCoresMilli(),
			"memory": p.MinMemMiB(),
		},
		"limits": map[string]any{
			"cpu":    p.MaxCoresMilli(),
			"memory": p.MaxMemMiB(),
		},
	}
}
