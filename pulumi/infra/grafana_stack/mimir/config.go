package mimir

import (
	"fmt"
)

// Config is the YAML-decoded mimir configuration.
type Config struct {
	Version string `yaml:"version"`
	Monolithic bool   `yaml:"monolithic"`
	Bucket string `yaml:"bucket"`
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	return nil
}
