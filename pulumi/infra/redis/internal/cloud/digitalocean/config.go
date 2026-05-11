package digitalocean

// Config is YAML under redis.cloud.digitalocean.
type Config struct{}

func (*Config) Validate() error {
	return nil
}
