// Package cloud holds YAML config for external object storage backends and dispatches
// validation to provider-specific packages under internal/cloud/<provider>.
package cloud

import (
	"fmt"

	"github.com/milagre/zote/pulumi/svc/objectstorage/internal/cloud/digitalocean"
)

// Config is YAML under objectstorage.cloud.
type Config struct {
	DigitalOcean *digitalocean.Config `yaml:"digitalocean"`
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("cloud config is nil")
	}

	if c.DigitalOcean != nil {
		return c.DigitalOcean.Validate()
	}

	return fmt.Errorf("no cloud object storage provider configured")
}
