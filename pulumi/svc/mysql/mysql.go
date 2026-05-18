// Package mysql deploys MySQL (in-cluster StatefulSet or DO managed cluster); same ConfigMap/Secret shape either way.
package mysql

import (
	"fmt"
	"strings"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/cloud"
	"github.com/milagre/zote/pulumi/svc/mysql/internal/container"
	"github.com/milagre/zote/pulumi/svc/mysql/internal/digitalocean"
	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/util/stringdata"
	"github.com/milagre/zote/pulumi/util/tokens"
)

var typeToken = tokens.Token("svc", "Mysql")

// Args bundles the YAML-decoded Config with the runtime identity
// (Env, Namespace, Name, Database, Username) and the multi-provider
// Cloud container. Cloud is consulted only when Config.Cloud is
// populated; the provider field matching Config.Cloud must be non-nil.
type Args struct {
	Env       env.Env
	Namespace string
	Name      string
	Database  string
	Username  string

	Config Config
	Cloud  cloud.Cloud
}

type K8s struct {
	ConfigMap pulumi.StringOutput
	Secret    pulumi.StringOutput
}

type Mysql struct {
	pulumi.ResourceState

	K8s K8s
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Mysql, error) {
	if args == nil {
		return nil, fmt.Errorf("%s: args is required", typeToken)
	}
	if err := args.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	resourceName := tokens.Qualify(args.Namespace, name)

	comp := &Mysql{}
	if err := ctx.RegisterComponentResource(typeToken, resourceName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	be, err := selectBackend(ctx, resourceName, args, comp)
	if err != nil {
		return nil, err
	}

	clientName := fmt.Sprintf("mysql-%s", args.Name)
	cfgPrefix := fmt.Sprintf("%s_MYSQL_%s", args.Env.Prefix, strings.ToUpper(args.Name))
	labels := pulumi.StringMap{"app": pulumi.String(clientName)}

	patchForce := pulumi.StringMap{"pulumi.com/patchForce": pulumi.String("true")}

	cm, err := corev1.NewConfigMap(ctx, resourceName, &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(clientName),
			Namespace:   pulumi.String(args.Namespace),
			Labels:      labels,
			Annotations: patchForce,
		},
		Data: pulumi.StringMap{
			cfgPrefix + "_HOST":     be.Hostname(),
			cfgPrefix + "_PORT":     be.Port(),
			cfgPrefix + "_USER":     be.Username(),
			cfgPrefix + "_DATABASE": pulumi.String(args.Database).ToStringOutput(),
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: client configmap: %w", typeToken, err)
	}

	sec, err := corev1.NewSecret(ctx, resourceName, &corev1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(clientName),
			Namespace:   pulumi.String(args.Namespace),
			Labels:      labels,
			Annotations: patchForce,
		},
		Type: pulumi.String("Opaque"),
		Data: stringdata.SecretData(map[string]pulumi.StringOutput{
			cfgPrefix + "_PASS": be.Password(),
		}),
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: client secret: %w", typeToken, err)
	}

	comp.K8s.ConfigMap = cm.Metadata.Name().Elem()
	comp.K8s.Secret = sec.Metadata.Name().Elem()

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"k8s": pulumi.Map{
			"configMap": comp.K8s.ConfigMap,
			"secret":    comp.K8s.Secret,
		},
	}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

type backend interface {
	Hostname() pulumi.StringOutput
	Port() pulumi.StringOutput
	Username() pulumi.StringOutput
	Password() pulumi.StringOutput
}

func selectBackend(ctx *pulumi.Context, name string, args *Args, parent pulumi.Resource) (backend, error) {
	switch {
	case args.Config.Container != nil:
		c, err := container.New(ctx, name, &container.Args{
			Env:       args.Env,
			Namespace: args.Namespace,
			Name:      fmt.Sprintf("mysql-%s", args.Name),
			Version:   args.Config.Version,
			Container: args.Config.Container,
			Database:  args.Database,
			Username:  args.Username,
		}, pulumi.Parent(parent))
		if err != nil {
			return nil, fmt.Errorf("%s: container backend: %w", typeToken, err)
		}

		return c, nil

	case args.Config.Cloud != nil && args.Config.Cloud.DigitalOcean != nil:
		d, err := digitalocean.New(ctx, name, &digitalocean.Args{
			Namespace: args.Namespace,
			Name:      fmt.Sprintf("mysql-%s", args.Name),
			Database:  args.Database,
			Version:   args.Config.Version,
			Cloud:     args.Cloud,
			Config:    args.Config.Cloud.DigitalOcean,
		}, pulumi.Parent(parent))
		if err != nil {
			return nil, fmt.Errorf("%s: digitalocean backend: %w", typeToken, err)
		}

		return d, nil
	}

	return nil, fmt.Errorf("%s: unreachable: args.validate() should have caught missing backend", typeToken)
}

func (a *Args) validate() error {
	if a.Name == "" {
		return fmt.Errorf("Name is required")
	}
	if a.Namespace == "" {
		return fmt.Errorf("Namespace is required")
	}
	if a.Database == "" {
		return fmt.Errorf("Database is required")
	}
	if a.Username == "" {
		return fmt.Errorf("Username is required")
	}
	if err := a.Env.Validate(); err != nil {
		return fmt.Errorf("invalid env: %w", err)
	}
	if err := a.Config.Validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if a.Config.Cloud != nil && a.Config.Cloud.DigitalOcean != nil &&
		a.Cloud.DigitalOcean == nil {
		return fmt.Errorf("Cloud.DigitalOcean is required when config.cloud.digitalocean is set")
	}

	return nil
}
