package loki

import (
	"fmt"

	"github.com/milagre/zote/pulumi/infra/grafana/loki/internal/digitalocean"
	"github.com/milagre/zote/pulumi/infra/grafana/loki/internal/local"
	"github.com/milagre/zote/pulumi/profile"
)

// Config is the YAML-decoded loki configuration. Exactly one of Cloud
// and Local must be populated; runtime cloud handles flow via
// Args.Cloud when Config.Cloud is set.
type Config struct {
	Version string      `yaml:"version"`
	Profile profile.Raw `yaml:"profile"`

	Cloud *Cloud      `yaml:"cloud"`
	Local *local.Spec `yaml:"local"`
}

type Cloud struct {
	DigitalOcean *digitalocean.Spec `yaml:"digitalocean"`
}

func (c *Config) Validate() error {
	if _, err := profile.New(c.Profile); err != nil {
		return fmt.Errorf("profile: %w", err)
	}

	switch {
	case c.Cloud != nil && c.Local != nil:
		return fmt.Errorf("cloud and local are mutually exclusive")

	case c.Cloud == nil && c.Local == nil:
		return fmt.Errorf("a storage backend is required (cloud or local)")
	}

	if c.Cloud != nil {
		if err := c.Cloud.Validate(); err != nil {
			return fmt.Errorf("cloud: %w", err)
		}
	}
	if c.Local != nil {
		if err := c.Local.Validate(); err != nil {
			return fmt.Errorf("local: %w", err)
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
