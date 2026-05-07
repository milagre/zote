package rabbitmq

import (
	"fmt"

	"github.com/milagre/zote/pulumi/infra/rabbitmq/internal/container"
)

// Config is the YAML-decoded rabbitmq configuration. Workload topology
// (users, vhosts) is not YAML-owned here — callers attach it via
// [Args.Setup] (e.g. nsinfra.RabbitmqSetup).
type Config struct {
	Version string `yaml:"version"`

	Container *container.Spec `yaml:"container"`
}

func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}
	if c.Container == nil {
		return fmt.Errorf("container is required")
	}
	if err := c.Container.Validate(); err != nil {
		return fmt.Errorf("container: %w", err)
	}

	return nil
}
