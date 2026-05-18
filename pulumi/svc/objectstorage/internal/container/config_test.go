package container_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/svc/objectstorage/internal/container"
	"github.com/milagre/zote/pulumi/util/profile"
)

func validProfile() *profile.Raw {
	return &profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "250m"},
		Mem: profile.RawRange{Min: "512M", Max: "1024M"},
	}
}

func TestConfig_Validate_ok(t *testing.T) {
	t.Parallel()

	c := &container.Config{
		Buckets: []container.Bucket{{Name: "b1"}},
		Profile: validProfile(),
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestConfig_Validate_profileNum(t *testing.T) {
	t.Parallel()

	raw := validProfile()
	raw.Num = &profile.IntRange{Min: 16, Max: 16}

	c := &container.Config{
		Buckets: []container.Bucket{{Name: "b1"}},
		Profile: raw,
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestConfig_Validate_errors(t *testing.T) {
	t.Parallel()

	cases := map[string]*container.Config{
		"nil":        nil,
		"no buckets": {Buckets: nil},
		"empty bucket name": {
			Buckets: []container.Bucket{{Name: ""}},
		},
		"invalid profile mem": {
			Buckets: []container.Bucket{{Name: "b1"}},
			Profile: &profile.Raw{
				CPU: profile.RawRange{Min: "100m", Max: "250m"},
				Mem: profile.RawRange{Min: "512Mi", Max: "1024M"},
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
