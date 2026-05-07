package redis

import (
	"fmt"

	"github.com/milagre/zote/pulumi/infra/redis/internal/container"
)

// Config is the YAML-decoded redis configuration. Only the container
// backend is implemented; the tree matches other data-tier components.
type Config struct {
	Version string `yaml:"version"`
	Shards  int    `yaml:"shards"`
	// Replicas is replicas per shard; StatefulSet size is Shards*(Replicas+1).
	Replicas  int             `yaml:"replicas"`
	Container *container.Spec `yaml:"container"`
}

func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}
	if c.Shards <= 0 {
		return fmt.Errorf("shards must be > 0")
	}
	if c.Replicas < 0 {
		return fmt.Errorf("replicas must be >= 0")
	}
	if c.Container == nil {
		return fmt.Errorf("container is required")
	}
	if err := c.Container.Validate(); err != nil {
		return fmt.Errorf("container: %w", err)
	}

	return nil
}
