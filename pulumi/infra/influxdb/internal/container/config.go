package container

import (
	"fmt"

	"github.com/milagre/zote/pulumi/profile"
)

// Spec is influxdb's YAML container branch (profile only today).
type Spec struct {
	Profile profile.Raw `yaml:"profile"`
}

func (s *Spec) Validate() error {
	if s == nil {
		return fmt.Errorf("spec is nil")
	}

	if _, err := profile.New(s.Profile); err != nil {
		return fmt.Errorf("profile: %w", err)
	}

	return nil
}
