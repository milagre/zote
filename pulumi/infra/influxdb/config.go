package influxdb

import (
	"fmt"

	"github.com/milagre/zote/pulumi/infra/influxdb/internal/container"
)

const (
	defaultOrg  = "influxdb"
	defaultUser = "admin"
)

// Config is the YAML-decoded influxdb configuration. Empty Organization
// or User become the package defaults at deploy time.
type Config struct {
	Version string `yaml:"version"`

	Organization string `yaml:"organization"`
	User         string `yaml:"user"`

	Container *container.Spec `yaml:"container"`
}

func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}
	if c.Container == nil {
		return fmt.Errorf("container is required")
	}
	if err := c.Container.Validate(); err != nil {
		return fmt.Errorf("container: %w", err)
	}

	return nil
}

// OrganizationOrDefault returns Organization or the built-in default org name.
func (c *Config) OrganizationOrDefault() string {
	if c.Organization != "" {
		return c.Organization
	}

	return defaultOrg
}

// UserOrDefault returns User or the built-in default admin user name.
func (c *Config) UserOrDefault() string {
	if c.User != "" {
		return c.User
	}

	return defaultUser
}
