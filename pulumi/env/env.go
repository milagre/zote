// Package env is deploy identity (type, tier, name, …) plus helpers for
// RandomPassword keepers and stack config ([WithRotateSecretsFromPulumi]).
package env

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	pulumiconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// RotateSecretsKeeperKey is merged into RandomPassword keepers when [Env.RotateSecrets] is set.
const RotateSecretsKeeperKey = "zote.rotateSecrets"

type Env struct {
	Type   string
	Tier   string
	Name   string
	Root   string
	Prefix string
	// Opaque stack bump string (e.g. zote:rotateSecrets); merged into random keepers when non-empty.
	RotateSecrets string
}

type Option func(*Env)

func WithRotateSecrets(v string) Option {
	return func(e *Env) { e.RotateSecrets = v }
}

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

// RandomKeepersOption configures [Env.RandomKeepers].
type RandomKeepersOption func(*randomKeepersOpts)

type randomKeepersOpts struct {
	supportsRotation bool
}

// SupportsRotation: when false, [Env.RotateSecrets] is not merged into keepers (default true).
func SupportsRotation(v bool) RandomKeepersOption {
	return func(o *randomKeepersOpts) {
		o.supportsRotation = v
	}
}

// RandomKeepers returns base plus RotateSecretsKeeperKey when rotation is enabled and [Env.RotateSecrets] is set.
func (e Env) RandomKeepers(base pulumi.StringMap, opts ...RandomKeepersOption) pulumi.StringMapInput {
	cfg := randomKeepersOpts{supportsRotation: true}
	for _, o := range opts {
		o(&cfg)
	}

	if !cfg.supportsRotation {
		return base
	}

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
