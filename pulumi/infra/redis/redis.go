// Package redis deploys Redis; host/port in a ConfigMap. Container backend only for now.
package redis

import (
	"fmt"
	"strings"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra/redis/internal/container"
	"github.com/milagre/zote/pulumi/tokens"
)

var typeToken = tokens.Token("infra", "Redis")

type ContainerArgs = container.Args

type Args struct {
	Env       env.Env
	Namespace string
	Name      string
	Container *ContainerArgs
}

type K8s struct {
	ConfigMap pulumi.StringOutput
}

type Redis struct {
	pulumi.ResourceState

	K8s K8s
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Redis, error) {
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

	comp := &Redis{}
	if err := ctx.RegisterComponentResource(typeToken, resourceName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	be, err := selectBackend(ctx, resourceName, args, comp)
	if err != nil {
		return nil, err
	}

	cfgPrefix := fmt.Sprintf("%s_REDIS_%s", args.Env.Prefix, cfgNameKey(args.Name))
	clientName := fmt.Sprintf("redis-%s", args.Name)

	cm, err := corev1.NewConfigMap(ctx, resourceName, &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(clientName),
			Namespace:   pulumi.String(args.Namespace),
			Annotations: pulumi.StringMap{"pulumi.com/patchForce": pulumi.String("true")},
		},
		Data: pulumi.StringMap{
			cfgPrefix + "_HOST": be.Hostname(),
			cfgPrefix + "_PORT": be.Port(),
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: client configmap: %w", typeToken, err)
	}

	comp.K8s.ConfigMap = cm.Metadata.Name().Elem()

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"k8s": pulumi.Map{
			"configMap": comp.K8s.ConfigMap,
		},
	}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

type backend interface {
	Hostname() pulumi.StringOutput
	Port() pulumi.StringOutput
}

func selectBackend(ctx *pulumi.Context, name string, args *Args, parent pulumi.Resource) (backend, error) {
	cArgs := *args.Container
	cArgs.Namespace = args.Namespace
	cArgs.Name = args.Name
	c, err := container.New(ctx, name, &cArgs, pulumi.Parent(parent))
	if err != nil {
		return nil, fmt.Errorf("%s: container backend: %w", typeToken, err)
	}

	return c, nil
}

func cfgNameKey(name string) string {
	return strings.ReplaceAll(strings.ToUpper(name), "-", "_")
}
