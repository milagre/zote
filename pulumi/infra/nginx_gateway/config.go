package nginx_gateway

import (
	"fmt"

	"github.com/milagre/zote/pulumi/util/profile"
)

// Config is YAML for NGINX Gateway Fabric. Empty Version uses the baked-in
// default; Profile sizes each Gateway's data plane.
type Config struct {
	Version string      `yaml:"version"`
	Profile profile.Raw `yaml:"profile"`

	// ExternalCRDs is set where the platform installs the Gateway API CRDs
	// itself, so they are not installed here.
	ExternalCRDs bool `yaml:"external_crds"`
}

func (c *Config) ResourceProfile() (profile.Profile, error) {
	p, err := profile.New(c.Profile)
	if err != nil {
		return profile.Profile{}, err
	}

	return p, nil
}

func (c *Config) Validate() error {
	if _, err := c.ResourceProfile(); err != nil {
		return fmt.Errorf("profile: %w", err)
	}

	return nil
}
