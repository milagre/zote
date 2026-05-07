package alloy

// Config is YAML for the alloy Helm chart. Empty Version uses the baked-in default.
type Config struct {
	Version string `yaml:"version"`
}

func (c *Config) Validate() error {
	return nil
}
