package container

import (
	"fmt"

	"github.com/milagre/zote/pulumi/util/profile"
)

// Spec is rabbitmq's YAML container branch (profile only; users/vhosts
// can be supplied programmatically via parent [rabbitmq.Args.Setup]).
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
