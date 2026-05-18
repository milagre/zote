package objectstorage_test

import (
	"testing"

	"github.com/milagre/zote/pulumi/svc/objectstorage"
	"github.com/milagre/zote/pulumi/svc/objectstorage/internal/container"
	"github.com/milagre/zote/pulumi/util/profile"
)

func validContainerProfile() *profile.Raw {
	return &profile.Raw{
		CPU: profile.RawRange{Min: "100m", Max: "250m"},
		Mem: profile.RawRange{Min: "512M", Max: "1024M"},
	}
}

func TestConfig_Validate_containerProfile_ok(t *testing.T) {
	t.Parallel()

	c := objectstorage.Config{
		Container: &container.Config{
			Buckets: []container.Bucket{{Name: "b1"}},
			Profile: validContainerProfile(),
		},
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestConfig_Validate_containerProfile_withNum(t *testing.T) {
	t.Parallel()

	raw := validContainerProfile()
	raw.Num = &profile.IntRange{Min: 16, Max: 16}

	c := objectstorage.Config{
		Container: &container.Config{
			Buckets: []container.Bucket{{Name: "b1"}},
			Profile: raw,
		},
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestConfig_Validate_containerProfile_errors(t *testing.T) {
	t.Parallel()

	cases := map[string]*objectstorage.Config{
		"invalid mem": {
			Container: &container.Config{
				Buckets: []container.Bucket{{Name: "b1"}},
				Profile: &profile.Raw{
					CPU: profile.RawRange{Min: "100m", Max: "250m"},
					Mem: profile.RawRange{Min: "512Mi", Max: "1024M"},
				},
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
