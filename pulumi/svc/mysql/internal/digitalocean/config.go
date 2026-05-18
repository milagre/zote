package digitalocean

import "fmt"

// Spec is mysql's YAML cloud.digitalocean branch.
type Spec struct {
	Primary  Primary   `yaml:"primary"`
	Replicas *Replicas `yaml:"replicas"`
}

func (s *Spec) Validate() error {
	if s == nil {
		return fmt.Errorf("spec is nil")
	}
	if s.Primary.Class == "" {
		return fmt.Errorf("primary.class is required")
	}
	if s.Replicas != nil && s.Replicas.Num > 0 && s.Replicas.Class == "" {
		return fmt.Errorf("replicas.class is required when replicas.num > 0")
	}

	return nil
}
