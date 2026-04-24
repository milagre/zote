// Package rabbitmq provides a ComponentResource that deploys a rabbitmq
// cluster and exposes its connection details (AMQP + management API) via
// ConfigMaps and per-user Secrets.
//
// The component is backend-polymorphic: today only the container backend
// is implemented, but the shape of Args leaves room for managed-cloud
// backends to be added without changing the client-facing resources.
package rabbitmq

import (
	"fmt"
	"sort"
	"strings"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra/rabbitmq/internal/container"
	"github.com/milagre/zote/pulumi/stringdata"
	"github.com/milagre/zote/pulumi/tokens"
)

var typeToken = tokens.Token("infra", "Rabbitmq")

// Caller-facing aliases for the in-cluster container backend. They are
// re-exports of the internal implementation types so callers can populate
// them without importing the internal package.
type (
	ContainerArgs  = container.Args
	ContainerSetup = container.Setup
	ContainerUser  = container.User
	ContainerVhost = container.Vhost
)

// Args configures a new Rabbitmq instance. Exactly one backend pointer
// must be non-nil (currently just Container).
type Args struct {
	Env       env.Env
	Namespace string
	Name      string

	// Container selects the in-cluster container backend. Mutually
	// exclusive with future cloud-backend pointers.
	Container *ContainerArgs
}

// Endpoint is a host/port pair for a network listener.
type Endpoint struct {
	Host pulumi.StringOutput
	Port pulumi.StringOutput
}

// K8sConfigMap is a single-ConfigMap grouping used for the non-user-scoped
// Kubernetes resources (management/API listener, AMQP listener). It is an
// object type rather than a bare string so sibling fields (e.g. a
// secret) can be added later without changing its position in the
// component's exposed surface.
type K8sConfigMap struct {
	ConfigMap pulumi.StringOutput
}

// K8sUser is the per-user client ConfigMap/Secret pair.
type K8sUser struct {
	ConfigMap pulumi.StringOutput
	Secret    pulumi.StringOutput
}

// K8s holds the names of the Kubernetes resources the component creates
// in the target namespace. Grouping the names under a struct (rather
// than flattening them onto the component) makes it obvious at the call
// site that the strings are in-cluster resource names, not arbitrary
// configuration values.
type K8s struct {
	Rabbitmq K8sConfigMap
	AMQP     K8sConfigMap
	Users    map[string]K8sUser
}

// Rabbitmq is the component resource. Its exposed surface is:
//
//   - K8s — names of the Kubernetes resources the component creates in
//     the target namespace.
//   - AMQP — AMQP listener endpoint (host/port).
//   - API — management API endpoint (host/port).
//   - Users — generated plaintext passwords keyed by username. Sensitive;
//     callers should pass these through Pulumi Secret wrappers when
//     persisting them.
type Rabbitmq struct {
	pulumi.ResourceState

	K8s   K8s
	AMQP  Endpoint
	API   Endpoint
	Users map[string]pulumi.StringOutput
}

// New registers the Rabbitmq component and its chosen backend.
func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Rabbitmq, error) {
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

	comp := &Rabbitmq{}
	if err := ctx.RegisterComponentResource(typeToken, resourceName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	be, err := selectBackend(ctx, resourceName, args, comp)
	if err != nil {
		return nil, err
	}

	cfgAMQP := fmt.Sprintf("%s_AMQP_%s", args.Env.Prefix, strings.ToUpper(args.Name))
	cfgRabbitmq := fmt.Sprintf("%s_RABBITMQ_%s", args.Env.Prefix, strings.ToUpper(args.Name))
	nameAMQP := fmt.Sprintf("amqp-%s", args.Name)
	nameRabbitmq := fmt.Sprintf("rabbitmq-%s", args.Name)

	patchForce := pulumi.StringMap{"pulumi.com/patchForce": pulumi.String("true")}

	amqpCM, err := corev1.NewConfigMap(ctx, resourceName+"-amqp", &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(nameAMQP),
			Namespace:   pulumi.String(args.Namespace),
			Annotations: patchForce,
		},
		Data: pulumi.StringMap{
			cfgAMQP + "_HOST": be.Hostname(),
			cfgAMQP + "_PORT": be.Port(),
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: amqp configmap: %w", typeToken, err)
	}

	rabbitmqCM, err := corev1.NewConfigMap(ctx, resourceName+"-rabbitmq", &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(nameRabbitmq),
			Namespace:   pulumi.String(args.Namespace),
			Annotations: patchForce,
		},
		Data: pulumi.StringMap{
			cfgRabbitmq + "_HOST": be.AdminHostname(),
			cfgRabbitmq + "_PORT": be.AdminPort(),
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: rabbitmq configmap: %w", typeToken, err)
	}

	users := make(map[string]K8sUser, len(args.Container.Setup.Users))
	for _, user := range sortedUsers(args.Container.Setup.Users) {
		cm, err := corev1.NewConfigMap(ctx, resourceName+"-user-"+user, &corev1.ConfigMapArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:        pulumi.String(nameAMQP + "-" + user),
				Namespace:   pulumi.String(args.Namespace),
				Annotations: patchForce,
			},
			Data: pulumi.StringMap{
				cfgAMQP + "_USER": pulumi.String(user),
			},
		}, pulumi.Parent(comp))
		if err != nil {
			return nil, fmt.Errorf("%s: user %q configmap: %w", typeToken, user, err)
		}

		pw, ok := be.Passwords()[user]
		if !ok {
			return nil, fmt.Errorf("%s: backend did not return password for user %q", typeToken, user)
		}
		sec, err := corev1.NewSecret(ctx, resourceName+"-user-"+user, &corev1.SecretArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:        pulumi.String(nameAMQP + "-" + user),
				Namespace:   pulumi.String(args.Namespace),
				Annotations: patchForce,
			},
			Type: pulumi.String("Opaque"),
			Data: stringdata.SecretData(map[string]pulumi.StringOutput{
				cfgAMQP + "_PASS": pw,
			}),
		}, pulumi.Parent(comp))
		if err != nil {
			return nil, fmt.Errorf("%s: user %q secret: %w", typeToken, user, err)
		}

		users[user] = K8sUser{
			ConfigMap: cm.Metadata.Name().Elem(),
			Secret:    sec.Metadata.Name().Elem(),
		}
	}

	comp.K8s = K8s{
		Rabbitmq: K8sConfigMap{ConfigMap: rabbitmqCM.Metadata.Name().Elem()},
		AMQP:     K8sConfigMap{ConfigMap: amqpCM.Metadata.Name().Elem()},
		Users:    users,
	}
	comp.AMQP = Endpoint{Host: be.Hostname(), Port: be.Port()}
	comp.API = Endpoint{Host: be.AdminHostname(), Port: be.AdminPort()}
	comp.Users = be.Passwords()

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

type backend interface {
	Hostname() pulumi.StringOutput
	Port() pulumi.StringOutput
	AdminHostname() pulumi.StringOutput
	AdminPort() pulumi.StringOutput
	Passwords() map[string]pulumi.StringOutput
}

func selectBackend(ctx *pulumi.Context, name string, args *Args, parent pulumi.Resource) (backend, error) {
	cArgs := *args.Container
	cArgs.Env = args.Env
	cArgs.Namespace = args.Namespace
	cArgs.Name = args.Name
	c, err := container.New(ctx, name, &cArgs, pulumi.Parent(parent))
	if err != nil {
		return nil, fmt.Errorf("%s: container backend: %w", typeToken, err)
	}

	return c, nil
}

// sortedUsers returns the user names in a deterministic order so the
// resource registration order is stable across Pulumi runs.
func sortedUsers(users []container.User) []string {
	out := make([]string, 0, len(users))
	for _, u := range users {
		out = append(out, u.Name)
	}
	sort.Strings(out)

	return out
}
