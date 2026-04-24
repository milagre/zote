// Package mysql provides a ComponentResource that deploys a mysql
// instance and exposes its connection details via a ConfigMap and
// Secret in the target namespace.
//
// The component is backend-polymorphic: a caller chooses between an
// in-cluster container backend (StatefulSet + PVC) and a managed
// DigitalOcean DatabaseCluster by supplying exactly one of Args.Container
// or Args.DigitalOcean. The set of client-facing Kubernetes resources
// the component emits is identical in both cases, so dependent workloads
// don't have to know which backend is live.
package mysql

import (
	"fmt"
	"strings"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/database/mysql/internal/container"
	"github.com/milagre/zote/pulumi/database/mysql/internal/digitalocean"
	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/stringdata"
	"github.com/milagre/zote/pulumi/tokens"
)

var typeToken = tokens.Token("database", "Mysql")

// Caller-facing aliases for the backend argument types. They are
// re-exports of the internal implementation types so callers can
// populate them without importing the internal packages. The Cloud
// field on DigitalOceanArgs is typed as database/digitalocean.Cloud;
// callers obtain a value satisfying it by calling
// cloud/digitalocean.Cloud.ForDatabase.
type (
	ContainerArgs        = container.Args
	DigitalOceanArgs     = digitalocean.Args
	DigitalOceanPrimary  = digitalocean.Primary
	DigitalOceanReplicas = digitalocean.Replicas
)

// Args configures a new Mysql instance. Exactly one of Container or
// DigitalOcean must be non-nil; the facade rejects configurations that
// select zero or more than one backend so the choice is explicit at
// the call site.
type Args struct {
	// Env is the deploy environment (used to derive the ConfigMap/Secret
	// key prefix).
	Env env.Env
	// Namespace is the target Kubernetes namespace for the shared
	// ConfigMap/Secret.
	Namespace string
	// Name is the logical mysql instance name (also part of the
	// ConfigMap/Secret key prefix).
	Name string
	// Version is the mysql engine version (backend-specific slug: an
	// image tag for Container, an API-accepted version string for
	// DigitalOcean).
	Version string
	// Database is the schema the instance is expected to host.
	Database string
	// Username is the non-root user seeded on the instance.
	Username string

	// Container selects the in-cluster StatefulSet backend. Mutually
	// exclusive with DigitalOcean.
	Container *ContainerArgs
	// DigitalOcean selects the managed DigitalOcean DatabaseCluster
	// backend. Mutually exclusive with Container. The caller populates
	// DigitalOceanArgs.Cloud with a per-instance handle returned by
	// cloud/digitalocean.Cloud.ForDatabase.
	DigitalOcean *DigitalOceanArgs
}

// K8s holds the names of the Kubernetes resources the component
// creates in the target namespace. Grouping the names under a struct
// (rather than flattening them onto the component) makes it obvious
// at the call site that the strings are in-cluster resource names,
// not arbitrary configuration values.
type K8s struct {
	ConfigMap pulumi.StringOutput
	Secret    pulumi.StringOutput
}

// Mysql is the component resource. Connection details are not exposed
// directly; they live in the ConfigMap and Secret named by K8s and are
// consumed by mounting those resources into dependent workloads.
type Mysql struct {
	pulumi.ResourceState

	K8s K8s
}

// New registers the Mysql component and its chosen backend.
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

// backend is the internal contract every backend implementation
// satisfies. Values flow directly into the shared ConfigMap/Secret, so
// everything is exposed as string outputs — scalar types (int ports)
// are stringified by the implementation.
type backend interface {
	Hostname() pulumi.StringOutput
	Port() pulumi.StringOutput
	Username() pulumi.StringOutput
	Password() pulumi.StringOutput
}

func selectBackend(ctx *pulumi.Context, name string, args *Args, parent pulumi.Resource) (backend, error) {
	switch {
	case args.Container != nil:
		cArgs := *args.Container
		cArgs.Env = args.Env
		cArgs.Namespace = args.Namespace
		cArgs.Name = fmt.Sprintf("mysql-%s", args.Name)
		cArgs.Version = args.Version
		cArgs.Database = args.Database
		cArgs.Username = args.Username
		c, err := container.New(ctx, name, &cArgs, pulumi.Parent(parent))
		if err != nil {
			return nil, fmt.Errorf("%s: container backend: %w", typeToken, err)
		}

		return c, nil

	case args.DigitalOcean != nil:
		dArgs := *args.DigitalOcean
		dArgs.Namespace = args.Namespace
		dArgs.Name = fmt.Sprintf("mysql-%s", args.Name)
		dArgs.Database = args.Database
		dArgs.Version = args.Version
		d, err := digitalocean.New(ctx, name, &dArgs, pulumi.Parent(parent))
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
	if a.Version == "" {
		return fmt.Errorf("Version is required")
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

	switch {
	case a.Container != nil && a.DigitalOcean != nil:
		return fmt.Errorf("Container and DigitalOcean are mutually exclusive")

	case a.Container == nil && a.DigitalOcean == nil:
		return fmt.Errorf("a backend is required (Container or DigitalOcean)")
	}

	return nil
}
