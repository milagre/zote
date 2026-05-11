// Package rabbitmq deploys RabbitMQ; AMQP/API ConfigMaps and per-user Secrets. Container backend only for now.
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
	"github.com/milagre/zote/pulumi/profile"
	"github.com/milagre/zote/pulumi/stringdata"
	"github.com/milagre/zote/pulumi/tokens"
)

var typeToken = tokens.Token("infra", "Rabbitmq")

type (
	ContainerSetup = container.Setup
	ContainerUser  = container.User
	ContainerVhost = container.Vhost
)

type Args struct {
	Env       env.Env
	Namespace string
	Name      string

	Config Config

	// Setup is the workload user/vhost topology (not YAML-decoded today).
	Setup container.Setup
}

type Endpoint struct {
	Host pulumi.StringOutput
	Port pulumi.StringOutput
}

type K8sConfigMap struct {
	ConfigMap pulumi.StringOutput
}

type K8sUser struct {
	ConfigMap pulumi.StringOutput
	Secret    pulumi.StringOutput
}

type K8s struct {
	Rabbitmq K8sConfigMap
	AMQP     K8sConfigMap
	Users    map[string]K8sUser
}

// Rabbitmq: K8s resource names; AMQP/API endpoints; Users map is plaintext passwords (treat as secret).
type Rabbitmq struct {
	pulumi.ResourceState

	K8s   K8s
	AMQP  Endpoint
	API   Endpoint
	Users map[string]pulumi.StringOutput
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Rabbitmq, error) {
	if args == nil {
		return nil, fmt.Errorf("%s: args is required", typeToken)
	}
	if err := args.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
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

	users := make(map[string]K8sUser, len(args.Setup.Users))
	for _, user := range sortedUsers(args.Setup.Users) {
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

func (a *Args) validate() error {
	if a.Name == "" {
		return fmt.Errorf("Name is required")
	}
	if a.Namespace == "" {
		return fmt.Errorf("Namespace is required")
	}
	if err := a.Env.Validate(); err != nil {
		return fmt.Errorf("invalid env: %w", err)
	}
	if err := a.Config.Validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	return nil
}

type backend interface {
	Hostname() pulumi.StringOutput
	Port() pulumi.StringOutput
	AdminHostname() pulumi.StringOutput
	AdminPort() pulumi.StringOutput
	Passwords() map[string]pulumi.StringOutput
}

func selectBackend(ctx *pulumi.Context, name string, args *Args, parent pulumi.Resource) (backend, error) {
	prof, err := profile.New(args.Config.Container.Profile)
	if err != nil {
		return nil, fmt.Errorf("%s: profile: %w", typeToken, err)
	}

	cArgs := container.Args{
		Env:       args.Env,
		Namespace: args.Namespace,
		Name:      args.Name,
		Version:   args.Config.Version,
		Profile:   prof,
		Setup:     args.Setup,
	}

	c, err := container.New(ctx, name, &cArgs, pulumi.Parent(parent))
	if err != nil {
		return nil, fmt.Errorf("%s: container backend: %w", typeToken, err)
	}

	return c, nil
}

func sortedUsers(users []container.User) []string {
	out := make([]string, 0, len(users))
	for _, u := range users {
		out = append(out, u.Name)
	}
	sort.Strings(out)

	return out
}
