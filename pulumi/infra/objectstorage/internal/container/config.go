package container

import (
	"fmt"

	"github.com/milagre/zote/pulumi/profile"
)

// Config is YAML under objectstorage.container for in-cluster MinIO.
type Config struct {
	Version string       `yaml:"version"`
	Size    *string      `yaml:"size"`
	User    string       `yaml:"user"`
	Buckets []Bucket     `yaml:"buckets"`
	Profile *profile.Raw `yaml:"profile,omitempty"`
}

// Bucket names a bucket the chart should create.
type Bucket struct {
	Name string `yaml:"name"`
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if len(c.Buckets) == 0 {
		return fmt.Errorf("buckets must contain at least one bucket")
	}
	for i, b := range c.Buckets {
		if b.Name == "" {
			return fmt.Errorf("buckets[%d].name is required", i)
		}
	}

	if c.Profile != nil {
		if _, err := profile.New(*c.Profile); err != nil {
			return fmt.Errorf("profile: %w", err)
		}
	}

	return nil
}
