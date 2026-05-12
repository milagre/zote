package loki

import (
	"fmt"

	"github.com/milagre/zote/pulumi/profile"
)

// Config is the YAML-decoded loki configuration.
type Config struct {
	Version    string      `yaml:"version"`
	Monolithic bool        `yaml:"monolithic"`
	Profile    profile.Raw `yaml:"profile"`
	Bucket     string      `yaml:"bucket"`
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if _, err := profile.New(c.Profile); err != nil {
		return fmt.Errorf("profile: %w", err)
	}
	if c.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}

	return nil
}
