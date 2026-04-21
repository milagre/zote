// Package env describes the deploy environment the rest of the library is
// being composed for. Core identity fields are plain Go values; RandomKeepers
// and WithRotateSecretsFromPulumi are the Pulumi-aware helpers so callers can
// wire RandomPassword keepers and stack config without importing pulumi at
// every component.
package env

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	pulumiconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// RotateSecretsKeeperKey is the keepers map key zote components use when
// Env.RotateSecrets is non-empty. The program injects RotateSecrets from its
// own configuration (e.g. Pulumi stack key zote:rotateSecrets); bumping that
// value forces replacement of random:index/randomPassword resources that
// merge this keeper.
const RotateSecretsKeeperKey = "zote.rotateSecrets"

type Env struct {
	Type   string
	Tier   string
	Name   string
	Root   string
	Prefix string
	// RotateSecrets is an opaque string (e.g. from Pulumi stack config via
	// WithRotateSecretsFromPulumi, or WithRotateSecrets for tests). When
	// non-empty, components that own RandomPassword resources merge it into
	// keepers under RotateSecretsKeeperKey so changing the value triggers
	// replacement — e.g. recovering empty result after a bad import.
	RotateSecrets string
}

// Option configures optional Env fields after the required identity tuple.
type Option func(*Env)

// WithRotateSecrets sets Env.RotateSecrets (see Env.RotateSecrets).
func WithRotateSecrets(v string) Option {
	return func(e *Env) { e.RotateSecrets = v }
}

// WithRotateSecretsFromPulumi reads the current stack's Pulumi config key
// zote:rotateSecrets and assigns it to Env.RotateSecrets (empty when unset).
func WithRotateSecretsFromPulumi(ctx *pulumi.Context) Option {
	return func(e *Env) {
		e.RotateSecrets = pulumiconfig.New(ctx, "zote").Get("rotateSecrets")
	}
}

func New(typ, tier, name, root, prefix string, opts ...Option) (Env, error) {
	e := Env{
		Type:   typ,
		Tier:   tier,
		Name:   name,
		Root:   root,
		Prefix: prefix,
	}
	for _, o := range opts {
		o(&e)
	}
	if err := e.Validate(); err != nil {
		return Env{}, fmt.Errorf("invalid env: %w", err)
	}

	return e, nil
}

func (e Env) Validate() error {
	if e.Type == "" {
		return fmt.Errorf("type is required")
	}
	if e.Tier == "" {
		return fmt.Errorf("tier is required")
	}
	if e.Name == "" {
		return fmt.Errorf("name is required")
	}
	if e.Root == "" {
		return fmt.Errorf("root is required")
	}

	return nil
}

func (e Env) ID() string {
	return e.Tier + "-" + e.Name
}

func (e Env) IsDev() bool {
	return e.Tier == "dev"
}

func (e Env) IsLocal() bool {
	return e.Type == "local"
}

func (e Env) LBType() string {
	if e.IsLocal() {
		return "NodePort"
	}

	return "LoadBalancer"
}

// RandomKeepers returns base unchanged when RotateSecrets is empty. Otherwise
// it returns a new StringMap with every entry from base plus
// RotateSecretsKeeperKey → RotateSecrets (base may be nil).
func (e Env) RandomKeepers(base pulumi.StringMap) pulumi.StringMapInput {
	if e.RotateSecrets == "" {
		return base
	}
	rot := pulumi.String(e.RotateSecrets)
	if len(base) == 0 {
		return pulumi.StringMap{RotateSecretsKeeperKey: rot}
	}
	out := pulumi.StringMap{}
	for k, v := range base {
		out[k] = v
	}
	out[RotateSecretsKeeperKey] = rot
	return out
}
