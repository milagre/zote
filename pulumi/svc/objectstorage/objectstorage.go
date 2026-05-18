// Package objectstorage provisions an S3-compatible object storage backend and
// publishes access details in a standard ConfigMap/Secret pair.
package objectstorage

import (
	"fmt"
	"strings"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/cloud"
	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/svc/objectstorage/internal/cloud/digitalocean"
	"github.com/milagre/zote/pulumi/svc/objectstorage/internal/container"
	"github.com/milagre/zote/pulumi/svc/objectstorage/internal/types"
	"github.com/milagre/zote/pulumi/util/stringdata"
	"github.com/milagre/zote/pulumi/util/tokens"
)

var typeToken = tokens.Token("svc", "ObjectStorage")

type Args struct {
	Env       env.Env
	Namespace string
	Name      string

	Config Config
	Cloud  cloud.Cloud
}

type K8s struct {
	ConfigMap pulumi.StringOutput
	Secret    pulumi.StringOutput
}

type Endpoint struct {
	URL  pulumi.StringOutput
	Host pulumi.StringOutput
	Port pulumi.StringOutput
}

// Addr returns host:port for S3 clients. Non-default ports (e.g. MinIO API on 9000) are otherwise omitted when only Host is set.
func (e Endpoint) Addr() pulumi.StringOutput {
	return pulumi.Sprintf("%s:%s", e.Host, e.Port)
}

type Credentials struct {
	AccessKey pulumi.StringOutput
	SecretKey pulumi.StringOutput
}

// ObjectStorage is the exported handle downstream components depend on.
// It is intentionally passed by value (no pointer required).
type ObjectStorage struct {
	// Deps are backend resources that must exist before consumers use this objectstorage. Pass to pulumi.DependsOn.
	Deps []pulumi.Resource

	K8s K8s

	S3 Endpoint

	Creds    Credentials
	Insecure pulumi.BoolOutput

	// Buckets maps each name from objectstorage buckets config to the provisioned
	// bucket name in the backend (identity for MinIO; prefixed for Spaces).
	Buckets map[string]string
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (ObjectStorage, error) {
	if args == nil {
		return ObjectStorage{}, fmt.Errorf("%s: args is required", typeToken)
	}
	if err := args.validate(); err != nil {
		return ObjectStorage{}, fmt.Errorf("%s: %w", typeToken, err)
	}

	resourceName := tokens.Qualify(args.Namespace, name)

	comp := &struct{ pulumi.ResourceState }{}
	if err := ctx.RegisterComponentResource(typeToken, resourceName, comp, opts...); err != nil {
		return ObjectStorage{}, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	cfgPrefix := fmt.Sprintf("%s_%s", args.Env.Prefix, strings.ToUpper(args.Name))
	clientName := fmt.Sprintf("objectstorage-%s-client", args.Name)
	labels := pulumi.StringMap{"app": pulumi.String(clientName)}

	out := ObjectStorage{}

	var res *types.Result
	var err error
	switch {
	case args.Config.Container != nil:
		cfg := args.Config.Container

		r, e := container.Setup(ctx, comp, &container.Args{
			Env:       args.Env,
			Namespace: args.Namespace,
			Name:      args.Name,
			Config:    cfg,
		})
		res, err = r, e

	case args.Config.Cloud != nil:
		cfg := args.Config.Cloud.DigitalOcean

		r, e := digitalocean.Setup(ctx, comp, &digitalocean.Args{
			Env:       args.Env,
			Cloud:     args.Cloud,
			Namespace: args.Namespace,
			Name:      args.Name,
			Config:    cfg,
		})
		res, err = r, e

	default:
		return ObjectStorage{}, fmt.Errorf("%s: dispatch: no backend selected", typeToken)
	}
	if err != nil {
		return ObjectStorage{}, fmt.Errorf("%s: backend: %w", typeToken, err)
	}

	cm, err := corev1.NewConfigMap(ctx, resourceName, &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(clientName),
			Namespace: pulumi.String(args.Namespace),
			Labels:    labels,
		},
		Data: pulumi.StringMap{
			cfgPrefix + "_S3_URL":  res.S3.URL,
			cfgPrefix + "_S3_HOST": res.S3.Host,
			cfgPrefix + "_S3_PORT": res.S3.Port,
		},
	}, pulumi.Parent(comp), pulumi.DependsOn(res.Deps))
	if err != nil {
		return ObjectStorage{}, fmt.Errorf("%s: configmap: %w", typeToken, err)
	}

	sec, err := corev1.NewSecret(ctx, resourceName, &corev1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(clientName),
			Namespace: pulumi.String(args.Namespace),
			Labels:    labels,
		},
		Type: pulumi.String("Opaque"),
		Data: stringdata.SecretData(map[string]pulumi.StringOutput{
			cfgPrefix + "_S3_ACCESS_KEY": res.Creds.AccessKey,
			cfgPrefix + "_S3_SECRET_KEY": res.Creds.SecretKey,
		}),
	}, pulumi.Parent(comp), pulumi.DependsOn([]pulumi.Resource{cm}))
	if err != nil {
		return ObjectStorage{}, fmt.Errorf("%s: secret: %w", typeToken, err)
	}

	out.K8s.ConfigMap = cm.Metadata.Name().Elem()
	out.K8s.Secret = sec.Metadata.Name().Elem()
	out.S3 = Endpoint(res.S3)
	out.Creds = Credentials(res.Creds)
	out.Insecure = res.Insecure
	out.Deps = res.Deps
	out.Buckets = res.Buckets

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"k8s": pulumi.Map{
			"configMap": out.K8s.ConfigMap,
			"secret":    out.K8s.Secret,
		},
	}); err != nil {
		return ObjectStorage{}, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return out, nil
}

// ProvisionedBucket returns the backend bucket name for a configured bucket name.
func (o ObjectStorage) ProvisionedBucket(configured string) (string, error) {
	if o.Buckets == nil {
		return "", fmt.Errorf("object storage has no bucket %q", configured)
	}

	s, ok := o.Buckets[configured]
	if !ok {
		return "", fmt.Errorf("object storage has no bucket %q", configured)
	}

	return s, nil
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

	if a.Config.Cloud != nil && a.Cloud.DigitalOcean == nil {
		return fmt.Errorf("Cloud.DigitalOcean is required when config.cloud is set")
	}

	return nil
}
