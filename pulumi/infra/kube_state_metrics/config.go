package kube_state_metrics

// Config is YAML for the kube-state-metrics chart. Empty Version uses ChartSpec.DefaultVersion.
type Config struct {
	Version string `yaml:"version"`
}

func (*Config) Validate() error {
	return nil
}
