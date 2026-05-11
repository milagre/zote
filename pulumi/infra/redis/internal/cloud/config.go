package cloud

import (
	"fmt"

	"github.com/milagre/zote/pulumi/infra/redis/internal/cloud/digitalocean"
)

// Config is YAML under redis.cloud.
type Config struct {
	DigitalOcean *digitalocean.Config `yaml:"digitalocean"`
}

func (c *Config) Validate() error {
	if c.DigitalOcean == nil {
		return fmt.Errorf("digitalocean is required when redis.cloud is set")
	}

	return c.DigitalOcean.Validate()
}
