package grafana

import "fmt"

// Config is YAML for the grafana chart. Empty Version uses the baked-in default.
type Config struct {
	Version string `yaml:"version"`
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	return nil
}
