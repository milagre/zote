// Package redis deploys Redis; host/port in a ConfigMap (container in-cluster or cloud managed).
package redis

import (
	"fmt"
	"strings"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/cloud"
	"github.com/milagre/zote/pulumi/env"
	doredis "github.com/milagre/zote/pulumi/infra/redis/internal/cloud/digitalocean"
	"github.com/milagre/zote/pulumi/infra/redis/internal/container"
	"github.com/milagre/zote/pulumi/profile"
	"github.com/milagre/zote/pulumi/tokens"
)

var typeToken = tokens.Token("infra", "Redis")

type Args struct {
	Env       env.Env
	Namespace string
	Name      string

	Config Config
	Cloud  cloud.Cloud
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
	if err := args.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
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

	redisType := clientRedisType(args.Config)

	cm, err := corev1.NewConfigMap(ctx, resourceName, &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(clientName),
			Namespace:   pulumi.String(args.Namespace),
			Annotations: pulumi.StringMap{"pulumi.com/patchForce": pulumi.String("true")},
		},
		Data: pulumi.StringMap{
			cfgPrefix + "_TYPE": pulumi.String(redisType),
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
}

func selectBackend(ctx *pulumi.Context, name string, args *Args, parent pulumi.Resource) (backend, error) {
	switch {
	case args.Config.Container != nil:
		prof, err := profile.New(args.Config.Container.Profile)
		if err != nil {
			return nil, fmt.Errorf("%s: profile: %w", typeToken, err)
		}

		cArgs := container.Args{
			Namespace: args.Namespace,
			Name:      args.Name,
			Version:   args.Config.Version,
			Profile:   prof,
			Shards:    args.Config.Shards,
			Replicas:  args.Config.Replicas,
			Standard:  containerRedisStandard(args.Config),
		}

		c, err := container.New(ctx, name, &cArgs, pulumi.Parent(parent))
		if err != nil {
			return nil, fmt.Errorf("%s: container backend: %w", typeToken, err)
		}

		return c, nil

	case args.Config.Cloud != nil:
		be, err := doredis.Setup(ctx, name, parent, &doredis.Args{
			Cloud:     args.Cloud,
			Namespace: args.Namespace,
			Name:      args.Name,
			Config:    args.Config.Cloud.DigitalOcean,
		})
		if err != nil {
			return nil, fmt.Errorf("%s: digitalocean backend: %w", typeToken, err)
		}

		return be, nil

	default:
		return nil, fmt.Errorf("%s: no backend selected", typeToken)
	}
}

func cfgNameKey(name string) string {
	return strings.ReplaceAll(strings.ToUpper(name), "-", "_")
}

func containerRedisStandard(cfg Config) bool {
	return cfg.Shards == 0 && cfg.Replicas == 0
}

// clientRedisType selects go-redis client mode for the app ConfigMap (see zredis.Aspect).
func clientRedisType(cfg Config) string {
	switch {
	case cfg.Cloud != nil:
		return cloudRedisClientType(cfg)
	case containerRedisStandard(cfg):
		return "standard"
	default:
		return "cluster"
	}
}

func cloudRedisClientType(cfg Config) string {
	_ = cfg
	// Managed Redis cluster vs standalone is provider-specific; wire when DO is implemented.
	return "cluster"
}
