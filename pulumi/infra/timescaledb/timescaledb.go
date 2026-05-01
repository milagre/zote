// Package timescaledb deploys TimescaleDB; ConfigMap + Secret. Container backend only for now.
package timescaledb

import (
	"fmt"
	"strings"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra/timescaledb/internal/container"
	"github.com/milagre/zote/pulumi/stringdata"
	"github.com/milagre/zote/pulumi/tokens"
)

var typeToken = tokens.Token("infra", "Timescaledb")

const (
	defaultUser     = "timescaledb"
	defaultDatabase = "timescaledb"
)

type ContainerArgs = container.Args

// Args: Container required. Empty User/Database → defaults below.
type Args struct {
	Env       env.Env
	Namespace string
	Name      string
	User      string
	Database  string
	Container *ContainerArgs
}

type K8s struct {
	ConfigMap pulumi.StringOutput
	Secret    pulumi.StringOutput
}

type Timescaledb struct {
	pulumi.ResourceState

	K8s K8s
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Timescaledb, error) {
	if args == nil {
		return nil, fmt.Errorf("%s: args is required", typeToken)
	}
	if args.Name == "" {
		return nil, fmt.Errorf("%s: Name is required", typeToken)
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("%s: Namespace is required", typeToken)
	}
	if err := args.Env.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}
	if args.Container == nil {
		return nil, fmt.Errorf("%s: a backend is required (currently only Container is implemented)", typeToken)
	}

	resourceName := tokens.Qualify(args.Namespace, name)

	comp := &Timescaledb{}
	if err := ctx.RegisterComponentResource(typeToken, resourceName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	be, err := selectBackend(ctx, resourceName, args, comp)
	if err != nil {
		return nil, err
	}

	cfgPrefix := fmt.Sprintf("%s_TIMESCALEDB_%s", args.Env.Prefix, strings.ToUpper(args.Name))
	clientName := fmt.Sprintf("timescaledb-%s", args.Name)
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
			cfgPrefix + "_SCHEME":   be.Scheme(),
			cfgPrefix + "_HOST":     be.Host(),
			cfgPrefix + "_PORT":     be.Port(),
			cfgPrefix + "_USER":     be.User(),
			cfgPrefix + "_DATABASE": be.Database(),
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
			cfgPrefix + "_PASSWORD": be.Pass(),
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
	Scheme() pulumi.StringOutput
	Host() pulumi.StringOutput
	Port() pulumi.StringOutput
	User() pulumi.StringOutput
	Pass() pulumi.StringOutput
	Database() pulumi.StringOutput
}

func selectBackend(ctx *pulumi.Context, name string, args *Args, parent pulumi.Resource) (backend, error) {
	user := args.User
	if user == "" {
		user = defaultUser
	}
	database := args.Database
	if database == "" {
		database = defaultDatabase
	}

	cArgs := *args.Container
	cArgs.Env = args.Env
	cArgs.Namespace = args.Namespace
	cArgs.Name = args.Name
	cArgs.User = user
	cArgs.Database = database
	c, err := container.New(ctx, name, &cArgs, pulumi.Parent(parent))
	if err != nil {
		return nil, fmt.Errorf("%s: container backend: %w", typeToken, err)
	}

	return c, nil
}
