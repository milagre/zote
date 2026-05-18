package container

import (
	"fmt"

	"github.com/milagre/zote/pulumi/util/profile"
)

// Spec is mysql's YAML container branch.
type Spec struct {
	Primary Tier `yaml:"primary"`
	Replica Tier `yaml:"replica"`
}

type Tier struct {
	Profile profile.Raw `yaml:"profile"`
}

func (s *Spec) Validate() error {
	if s == nil {
		return fmt.Errorf("spec is nil")
	}
	if _, err := profile.New(s.Primary.Profile); err != nil {
		return fmt.Errorf("primary.profile: %w", err)
	}
	if _, err := profile.New(s.Replica.Profile); err != nil {
		return fmt.Errorf("replica.profile: %w", err)
	}

	return nil
}
