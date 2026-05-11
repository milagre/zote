package digitalocean_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/infra/objectstorage/internal/cloud/digitalocean"
)

func TestConfig_Validate_ok(t *testing.T) {
	t.Parallel()

	c := &digitalocean.Config{
		Region:  "nyc3",
		Buckets: []digitalocean.Bucket{{Name: "a"}, {Name: "b"}},
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestConfig_Validate_errors(t *testing.T) {
	t.Parallel()

	cases := map[string]*digitalocean.Config{
		"nil":           nil,
		"no buckets":    {Region: "nyc3", Buckets: nil},
		"blank name":    {Region: "nyc3", Buckets: []digitalocean.Bucket{{Name: ""}}},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := c.Validate(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
