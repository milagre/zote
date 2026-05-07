package cert_manager

// Config is YAML for the cert-manager chart. Issuer identity still comes from
// root.acme.email (wired at deploy time).
type Config struct {
	Version string `yaml:"version"`
}

func (*Config) Validate() error {
	return nil
}
