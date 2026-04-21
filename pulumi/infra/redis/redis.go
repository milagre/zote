// Package redis provides a ComponentResource that deploys a redis cluster
// and exposes its connection details (host + port) via a ConfigMap.
//
// The component is backend-polymorphic: today only the container backend
// is implemented, but the shape of Args leaves room for managed-cloud
// backends to be added without changing the client-facing resources.
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

// ContainerArgs is the caller-facing configuration for the in-cluster
// container backend. It is a type alias over the internal implementation
// type so callers can populate it without importing the internal package.
type ContainerArgs = container.Args

// Args configures a new Redis instance. Exactly one backend pointer
// must be non-nil (currently just Container).
type Args struct {
	Env       env.Env
	Namespace string
	Name      string

	// Container selects the in-cluster container backend. Mutually
	// exclusive with future cloud-backend pointers.
	Container *ContainerArgs
}

// K8s holds the names of the Kubernetes resources the component creates
// in the target namespace. Grouping the names under a struct (rather than
// flattening them onto the component) makes it obvious at the call site
// that the strings are in-cluster resource names, not arbitrary
// configuration values.
type K8s struct {
	ConfigMap pulumi.StringOutput
}

// Redis is the component resource. Connection details are not exposed
// directly; they live in the ConfigMap named by K8s and are consumed by
// mounting that resource into dependent workloads.
type Redis struct {
	pulumi.ResourceState

	K8s K8s
}

// New registers the Redis component and its chosen backend.
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

// cfgNameKey renders a redis instance name as an environment-variable key
// fragment: upper-cased with dashes converted to underscores so the
// result is a valid shell identifier.
func cfgNameKey(name string) string {
	return strings.ReplaceAll(strings.ToUpper(name), "-", "_")
}
