package local

import "fmt"

// Spec is YAML config.local (SingleBinary on a PV in dev).
type Spec struct {
	Size *string `yaml:"size"`
}

func (s *Spec) Validate() error {
	if s == nil {
		return fmt.Errorf("spec is nil")
	}

	return nil
}
