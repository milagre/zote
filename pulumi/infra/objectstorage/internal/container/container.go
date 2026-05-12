// Package container provisions in-cluster MinIO and returns S3-compatible access details.
package container

import (
	"fmt"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra/objectstorage/internal/types"
	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/profile"
	"github.com/milagre/zote/pulumi/stringdata"
)

type Args struct {
	Env       env.Env
	Namespace string
	Name      string

	Config *Config
}

const (
	defaultSize              = "20Gi"
	defaultMinIOChartVersion = "5.4.0"
)

func Setup(ctx *pulumi.Context, parent pulumi.Resource, a *Args) (*types.Result, error) {
	if a == nil {
		return nil, fmt.Errorf("args is required")
	}
	if err := a.Env.Validate(); err != nil {
		return nil, fmt.Errorf("env: %w", err)
	}
	if a.Namespace == "" {
		return nil, fmt.Errorf("Namespace is required")
	}
	if a.Name == "" {
		return nil, fmt.Errorf("Name is required")
	}
	if a.Config == nil {
		return nil, fmt.Errorf("Config is required")
	}
	if err := a.Config.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	cfg := a.Config

	var prof *profile.Profile
	if cfg.Profile != nil {
		p, err := profile.New(*cfg.Profile)
		if err != nil {
			return nil, fmt.Errorf("profile: %w", err)
		}

		prof = &p
	}

	release := fmt.Sprintf("objectstorage-%s", a.Name)
	authSecretName := release + "-auth"
	host := fmt.Sprintf("%s.%s.svc.cluster.local", release, a.Namespace)

	pw, err := random.NewRandomPassword(ctx, a.Name+"-minio-password", &random.RandomPasswordArgs{
		Length:          pulumi.Int(48),
		Numeric:         pulumi.Bool(true),
		Upper:           pulumi.Bool(true),
		Lower:           pulumi.Bool(true),
		Special:         pulumi.Bool(false),
		MinNumeric:      pulumi.Int(8),
		MinLower:        pulumi.Int(8),
		MinUpper:        pulumi.Int(8),
		OverrideSpecial: pulumi.String("$%&*()-_=+[]{}<>:?"),
		Keepers:         a.Env.RandomKeepers(nil),
	}, pulumi.Parent(parent), pulumi.IgnoreChanges([]string{"*"}))
	if err != nil {
		return nil, fmt.Errorf("minio password: %w", err)
	}

	rootUser := cfg.User
	if rootUser == "" {
		rootUser = "minio"
	}

	sec, err := corev1.NewSecret(ctx, authSecretName, &corev1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(authSecretName),
			Namespace: pulumi.String(a.Namespace),
		},
		Type: pulumi.String("Opaque"),
		Data: stringdata.SecretData(map[string]pulumi.StringOutput{
			"rootUser":     pulumi.String(rootUser).ToStringOutput(),
			"rootPassword": pw.Result,
		}),
	}, pulumi.Parent(parent))
	if err != nil {
		return nil, fmt.Errorf("minio auth secret: %w", err)
	}

	size := defaultSize
	if cfg.Size != nil {
		size = *cfg.Size
	}

	finalBuckets := make(map[string]string, len(cfg.Buckets))
	buckets := make([]any, 0, len(cfg.Buckets))
	for _, b := range cfg.Buckets {
		finalBuckets[b.Name] = b.Name
		buckets = append(buckets, map[string]any{
			"name":       b.Name,
			"policy":     "none",
			"purge":      false,
			"versioning": false,
		})
	}

	// Chart defaults are mode=distributed and replicas=16; in-cluster object storage
	// for this stack is a single MinIO server (standalone, one StatefulSet replica).
	// fullnameOverride makes Service/StatefulSet names match the Helm release name
	// (no default "{release}-minio" suffix).
	vals := map[string]any{
		"fullnameOverride": release,
		"mode":             "standalone",
		"replicas":         1,
		"existingSecret":   authSecretName,
		"persistence": map[string]any{
			"enabled": true,
			"size":    size,
		},
		"buckets": buckets,
	}
	if prof != nil {
		p := prof
		vals["resources"] = map[string]any{
			"requests": map[string]any{
				"cpu":    p.MinCoresMilli(),
				"memory": p.MinMemMiB(),
			},
			"limits": map[string]any{
				"cpu":    p.MaxCoresMilli(),
				"memory": p.MaxMemMiB(),
			},
		}
	}
	values := helm.Values(vals)

	var ver pulumi.StringPtrInput
	switch v := helm.OptionalChartVersion(cfg.Version); {
	case v != nil:
		ver = pulumi.String(*v).ToStringPtrOutput()
	default:
		ver = pulumi.String(defaultMinIOChartVersion).ToStringPtrOutput()
	}

	rel, err := helmv3.NewRelease(ctx, release, &helmv3.ReleaseArgs{
		Chart:     pulumi.String("minio"),
		Name:      pulumi.String(release).ToStringPtrOutput(),
		Namespace: pulumi.String(a.Namespace).ToStringPtrOutput(),
		Version:   ver,
		RepositoryOpts: &helmv3.RepositoryOptsArgs{
			Repo: pulumi.String("https://charts.min.io/").ToStringPtrOutput(),
		},
		Values: values,
	}, pulumi.Parent(parent), pulumi.DependsOn([]pulumi.Resource{sec}))
	if err != nil {
		return nil, fmt.Errorf("minio helm release: %w", err)
	}

	return &types.Result{
		S3: types.Endpoint{
			URL:  pulumi.String(fmt.Sprintf("http://%s:9000", host)).ToStringOutput(),
			Host: pulumi.String(host).ToStringOutput(),
			Port: pulumi.String("9000").ToStringOutput(),
		},
		Creds: types.Credentials{
			AccessKey: pulumi.String(rootUser).ToStringOutput(),
			SecretKey: pw.Result,
		},
		Insecure: pulumi.Bool(true).ToBoolOutput(),
		Buckets:  finalBuckets,
		Deps:     []pulumi.Resource{sec, rel},
	}, nil
}
