package prometheus

// Config is YAML for kube-prometheus-stack. Empty Version uses ChartSpec.DefaultVersion.
type Config struct {
	Version string `yaml:"version"`
}

func (*Config) Validate() error {
	return nil
}
