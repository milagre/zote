package grafana_stack

import (
	"fmt"

	"github.com/milagre/zote/pulumi/infra/grafana_stack/alloy"
	"github.com/milagre/zote/pulumi/infra/grafana_stack/grafana"
	"github.com/milagre/zote/pulumi/infra/grafana_stack/loki"
	"github.com/milagre/zote/pulumi/infra/grafana_stack/mimir"
)

// Config is the YAML-decoded grafana_stack block. Each subsection is owned and
// validated by its subpackage (compare objectstorage container / cloud).
type Config struct {
	Dashboard *grafana.Config `yaml:"dashboard,omitempty"`
	Alloy     *alloy.Config   `yaml:"alloy,omitempty"`
	Loki      *loki.Config    `yaml:"loki,omitempty"`
	Mimir     *mimir.Config   `yaml:"mimir,omitempty"`
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.Dashboard == nil {
		return fmt.Errorf("dashboard is required")
	}
	if err := c.Dashboard.Validate(); err != nil {
		return fmt.Errorf("dashboard: %w", err)
	}

	if c.Alloy == nil {
		return fmt.Errorf("alloy is required")
	}
	if err := c.Alloy.Validate(); err != nil {
		return fmt.Errorf("alloy: %w", err)
	}

	if c.Loki == nil {
		return fmt.Errorf("loki is required")
	}
	if err := c.Loki.Validate(); err != nil {
		return fmt.Errorf("loki: %w", err)
	}

	if c.Mimir == nil {
		return fmt.Errorf("mimir is required")
	}
	if err := c.Mimir.Validate(); err != nil {
		return fmt.Errorf("mimir: %w", err)
	}

	return nil
}
