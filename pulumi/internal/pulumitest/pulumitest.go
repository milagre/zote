// Package pulumitest runs a Pulumi program against mocks and reports the
// resources it registered.
package pulumitest

import (
	"sync"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Resource is one registered resource, with its inputs as plain Go values.
type Resource struct {
	Type   string
	Name   string
	Inputs map[string]any
}

// Run runs program and returns every resource it registered, in order.
func Run(t testing.TB, program pulumi.RunFunc) []Resource {
	t.Helper()

	m := &mocks{}

	err := pulumi.RunErr(program, pulumi.WithMocks("project", "stack", m))
	if err != nil {
		t.Fatalf("running program: %v", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]Resource(nil), m.resources...)
}

// OfType returns the resources of type typ.
func OfType(rs []Resource, typ string) []Resource {
	var out []Resource
	for _, r := range rs {
		if r.Type == typ {
			out = append(out, r)
		}
	}

	return out
}

// Lookup walks keys through nested maps, returning nil where one is missing.
func Lookup(v any, keys ...string) any {
	for _, k := range keys {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}

	return v
}

type mocks struct {
	mu        sync.Mutex
	resources []Resource
}

func (m *mocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.resources = append(m.resources, Resource{
		Type:   args.TypeToken,
		Name:   args.Name,
		Inputs: args.Inputs.Mappable(),
	})

	return args.Name + "-id", args.Inputs, nil
}

func (m *mocks) Call(pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return resource.PropertyMap{}, nil
}
