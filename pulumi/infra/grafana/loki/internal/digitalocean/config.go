package digitalocean

import "fmt"

// Spec is YAML config.cloud.digitalocean.
type Spec struct {
	// Region defaults to the VPC's region when empty.
	Region string `yaml:"region"`
}

func (s *Spec) Validate() error {
	if s == nil {
		return fmt.Errorf("spec is nil")
	}

	return nil
}
