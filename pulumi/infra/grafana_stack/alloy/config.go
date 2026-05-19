package alloy

import (
	"fmt"

	"github.com/milagre/zote/pulumi/util/profile"
)

// Config is YAML-decoded Helm chart knobs for Alloy. Empty Version uses the baked-in default.
type Config struct {
	Version string      `yaml:"version"`
	Profile profile.Raw `yaml:"profile,omitempty"`
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.profileSet() {
		if _, err := profile.New(c.Profile); err != nil {
			return fmt.Errorf("profile: %w", err)
		}
	}

	return nil
}
