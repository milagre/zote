// Package digitalocean provisions DigitalOcean managed Redis (stub).
package digitalocean

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/cloud"
)

// Args configures the DigitalOcean Redis backend.
type Args struct {
	Cloud cloud.Cloud

	Namespace string
	Name      string

	Config *Config
}

// Backend publishes host/port for the root Redis component ConfigMap.
type Backend struct {
	host pulumi.StringOutput
	port pulumi.StringOutput
}

func (b *Backend) Hostname() pulumi.StringOutput { return b.host }
func (b *Backend) Port() pulumi.StringOutput     { return b.port }

// Setup provisions managed Redis. Not implemented yet.
func Setup(ctx *pulumi.Context, parentName string, parent pulumi.Resource, a *Args) (*Backend, error) {
	_ = ctx
	_ = parentName
	_ = parent

	if a == nil {
		return nil, fmt.Errorf("args is required")
	}
	if a.Namespace == "" {
		return nil, fmt.Errorf("Namespace is required")
	}
	if a.Name == "" {
		return nil, fmt.Errorf("Name is required")
	}
	if a.Config == nil {
		return nil, fmt.Errorf("Config is required")
	}
	if err := a.Config.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if a.Cloud.DigitalOcean == nil {
		return nil, fmt.Errorf("Cloud.DigitalOcean is required")
	}

	return nil, fmt.Errorf("digitalocean redis: not implemented")
}
