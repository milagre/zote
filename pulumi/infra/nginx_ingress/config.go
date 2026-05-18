package nginx_ingress

import (
	"fmt"

	"github.com/milagre/zote/pulumi/util/profile"
)

// DefaultIngressClass is controller.ingressClassResource.name for this chart.
const DefaultIngressClass = "nginx"

// Config drives the ingress-nginx Helm release (sizing profile + optional version).
type Config struct {
	Version string `yaml:"version"`

	Profile profile.Raw `yaml:"profile"`
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
