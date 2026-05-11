package alloy

import "fmt"

// Config is YAML-decoded Helm chart knobs for Alloy. Empty Version uses the baked-in default.
type Config struct {
	Version string `yaml:"version"`
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	return nil
}
