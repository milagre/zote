package mysql

import (
	"fmt"

	"github.com/milagre/zote/pulumi/svc/mysql/internal/container"
	"github.com/milagre/zote/pulumi/svc/mysql/internal/digitalocean"
)

// Config is the YAML-decoded mysql configuration. Exactly one of
// Cloud and Container must be populated; runtime cloud SDK handles
// flow separately via Args.Cloud.
type Config struct {
	Version string `yaml:"version"`

	Cloud     *Cloud          `yaml:"cloud"`
	Container *container.Spec `yaml:"container"`
}

type Cloud struct {
	DigitalOcean *digitalocean.Spec `yaml:"digitalocean"`
}

func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}

	switch {
	case c.Cloud != nil && c.Container != nil:
		return fmt.Errorf("cloud and container are mutually exclusive")

	case c.Cloud == nil && c.Container == nil:
		return fmt.Errorf("a backend is required (cloud or container)")
	}

	if c.Cloud != nil {
		if err := c.Cloud.Validate(); err != nil {
			return fmt.Errorf("cloud: %w", err)
		}
	}
	if c.Container != nil {
		if err := c.Container.Validate(); err != nil {
			return fmt.Errorf("container: %w", err)
		}
	}

	return nil
}

func (c *Cloud) Validate() error {
	if c.DigitalOcean == nil {
		return fmt.Errorf("at least one provider is required (digitalocean)")
	}
	if err := c.DigitalOcean.Validate(); err != nil {
		return fmt.Errorf("digitalocean: %w", err)
	}

	return nil
}
