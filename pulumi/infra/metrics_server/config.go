package metrics_server

// Config is YAML for the metrics-server chart. Empty Version uses ChartSpec.DefaultVersion.
type Config struct {
	Version string `yaml:"version"`
}

func (*Config) Validate() error {
	return nil
}
