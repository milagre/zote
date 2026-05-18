package objectstorage

import (
	"fmt"

	storcloud "github.com/milagre/zote/pulumi/svc/objectstorage/internal/cloud"
	"github.com/milagre/zote/pulumi/svc/objectstorage/internal/container"
)

type Config struct {
	Container *container.Config `yaml:"container"`
	Cloud     *storcloud.Config `yaml:"cloud"`
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	hasContainer := c.Container != nil
	hasCloud := c.Cloud != nil
	if hasContainer == hasCloud {
		return fmt.Errorf("exactly one of container or cloud is required")
	}

	if c.Container != nil {
		if err := c.Container.Validate(); err != nil {
			return fmt.Errorf("container: %w", err)
		}
	}

	if c.Cloud != nil {
		if err := c.Cloud.Validate(); err != nil {
			return fmt.Errorf("cloud: %w", err)
		}
	}

	return nil
}
