package cloudflare_tunnel

// Config is YAML for the cloudflare-tunnel-ingress-controller chart.
// Credentials stay on root.cloudflare.
type Config struct {
	Version string `yaml:"version"`
}

func (*Config) Validate() error {
	return nil
}
