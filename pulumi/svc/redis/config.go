package redis

import (
	"fmt"

	rediscloud "github.com/milagre/zote/pulumi/svc/redis/internal/cloud"
	"github.com/milagre/zote/pulumi/svc/redis/internal/container"
)

// Config is the YAML-decoded redis configuration. Exactly one of container or cloud.
type Config struct {
	Version string `yaml:"version"`
	// Shards and Replicas: omit both (YAML absent → zero) for single-node standard Redis; otherwise clustered.
	// Used only when container is set.
	Shards int `yaml:"shards"`
	// Replicas is replicas per shard; StatefulSet size is Shards*(Replicas+1). Container only.
	Replicas  int                `yaml:"replicas"`
	Container *container.Spec    `yaml:"container"`
	Cloud     *rediscloud.Config `yaml:"cloud"`
}

func (c *Config) Validate() error {
	hasContainer := c.Container != nil
	hasCloud := c.Cloud != nil
	if hasContainer == hasCloud {
		return fmt.Errorf("exactly one of container or cloud is required")
	}

	if c.Cloud != nil {
		return c.Cloud.Validate()
	}

	return c.validateContainer()
}

func (c *Config) validateContainer() error {
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}
	if c.Replicas < 0 {
		return fmt.Errorf("replicas must be >= 0")
	}
	standard := c.Shards == 0 && c.Replicas == 0
	if !standard && c.Shards <= 0 {
		return fmt.Errorf("shards must be > 0 for clustered redis (omit shards and replicas for single-node)")
	}
	if err := c.Container.Validate(); err != nil {
		return fmt.Errorf("container: %w", err)
	}

	return nil
}
