// Package distributed installs Mimir via the upstream mimir-distributed Helm chart against external S3.
package distributed

import (
	"fmt"
	"net/url"

	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/endpoint"
	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra/objectstorage"
	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/tokens"
)

const (
	chart          = "mimir-distributed"
	repository     = "https://grafana.github.io/helm-charts"
	defaultVersion = "6.1.0-weekly.391"
)

type Result struct {
	Release    *helmv3.Release
	Gateway    url.URL
	Prometheus url.URL
	Push       url.URL
}

type Args struct {
	Env           env.Env
	Namespace     string
	Name          string
	Bucket        string
	ChartVersion  string // empty uses defaultVersion
	ObjectStorage objectstorage.ObjectStorage
}

func Deploy(ctx *pulumi.Context, parent pulumi.Resource, a *Args) (*Result, error) {
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
	if a.Bucket == "" {
		return nil, fmt.Errorf("Bucket is required")
	}

	s3Bucket, err := a.ObjectStorage.ProvisionedBucket(a.Bucket)
	if err != nil {
		return nil, fmt.Errorf("bucket: %w", err)
	}

	resourceName := tokens.Qualify(a.Namespace, a.Name)

	values := helm.Values(map[string]any{
		"minio": map[string]any{"enabled": false},
		"mimir": map[string]any{
			"structuredConfig": map[string]any{
				"common": map[string]any{
					"storage": map[string]any{
						"backend": "s3",
						"s3": map[string]any{
							"endpoint":          a.ObjectStorage.S3.Addr(),
							"access_key_id":     a.ObjectStorage.Creds.AccessKey,
							"secret_access_key": a.ObjectStorage.Creds.SecretKey,
							"insecure":          a.ObjectStorage.Insecure,
							"bucket_name":       s3Bucket,
						},
					},
				},
				"blocks_storage": map[string]any{
					"storage_prefix": "blocks",
				},
				"ruler_storage": map[string]any{
					"storage_prefix": "ruler",
				},
				"alertmanager_storage": map[string]any{
					"storage_prefix": "alertmanager",
				},
			},
		},
	})

	ver := pulumi.String(defaultVersion).ToStringPtrOutput()
	if a.ChartVersion != "" {
		ver = pulumi.String(a.ChartVersion).ToStringPtrOutput()
	}

	relOpts := []pulumi.ResourceOption{pulumi.Parent(parent)}
	if d := a.ObjectStorage.Deps; len(d) > 0 {
		relOpts = append(relOpts, pulumi.DependsOn(d))
	}

	rel, err := helmv3.NewRelease(ctx, resourceName, &helmv3.ReleaseArgs{
		Chart:     pulumi.String(chart),
		Name:      pulumi.String(a.Name).ToStringPtrOutput(),
		Namespace: pulumi.String(a.Namespace).ToStringPtrOutput(),
		Version:   ver,
		RepositoryOpts: &helmv3.RepositoryOptsArgs{
			Repo: pulumi.String(repository).ToStringPtrOutput(),
		},
		Values: values,
	}, relOpts...)
	if err != nil {
		return nil, fmt.Errorf("helm release: %w", err)
	}

	gw, prom, push := mimirNginxURLs(a.Namespace, a.Name)
	return &Result{
		Release:    rel,
		Gateway:    gw,
		Prometheus: prom,
		Push:       push,
	}, nil
}

func mimirNginxURLs(namespace, release string) (gateway, prometheus, push url.URL) {
	host := fmt.Sprintf("%s-nginx.%s.svc.cluster.local", release, namespace)
	gateway = endpoint.HTTP(host, "80", "/")
	prometheus = endpoint.HTTP(host, "80", "/prometheus")
	push = endpoint.HTTP(host, "80", "/api/v1/push")
	return gateway, prometheus, push
}
