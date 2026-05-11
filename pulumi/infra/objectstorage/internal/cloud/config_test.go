package cloud_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/infra/objectstorage/internal/cloud"
	"github.com/milagre/zote/pulumi/infra/objectstorage/internal/cloud/digitalocean"
)

func TestConfig_Validate_digitalocean_ok(t *testing.T) {
	t.Parallel()

	c := &cloud.Config{
		DigitalOcean: &digitalocean.Config{
			Region:  "nyc3",
			Buckets: []digitalocean.Bucket{{Name: "a"}, {Name: "b"}},
		},
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestConfig_Validate_errors(t *testing.T) {
	t.Parallel()

	cases := map[string]*cloud.Config{
		"nil":         nil,
		"no provider": {},
		"missing digitalocean buckets": {
			DigitalOcean: &digitalocean.Config{Region: "nyc3", Buckets: nil},
		},
		"blank bucket name": {
			DigitalOcean: &digitalocean.Config{
				Region:  "nyc3",
				Buckets: []digitalocean.Bucket{{Name: ""}},
			},
		},
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
