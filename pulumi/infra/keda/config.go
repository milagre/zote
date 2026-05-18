package keda

// Config is YAML for the KEDA chart. Empty Version uses the baked-in default.
type Config struct {
	Version string `yaml:"version"`
}

func (*Config) Validate() error {
	return nil
}
