// Package monolithic deploys single-process Mimir (target=all) on Kubernetes.
package monolithic

import (
	"fmt"
	"net/url"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/svc/objectstorage"
	"github.com/milagre/zote/pulumi/util/annotations"
	"github.com/milagre/zote/pulumi/util/endpoint"
)

const defaultImage = "grafana/mimir:2.17.11"

type Result struct {
	Gateway    url.URL
	Prometheus url.URL
	Push       url.URL
	Deployment *appsv1.Deployment
	Service    *corev1.Service
}

type Args struct {
	Env       env.Env
	Namespace string
	Name      string

	ObjectStorage objectstorage.ObjectStorage

	Bucket string
	Image  *string
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

	image := defaultImage
	if a.Image != nil {
		image = *a.Image
	}

	// Monolithic config: single bucket with separate prefixes.
	// ObjectStorage bucket and credentials are passed via env expansion.
	// Default Mimir replication factor is 3; this deployment runs one pod (replicas: 1). Without RF=1
	// Grafana shows "too many unhealthy instances in the ring" while query-frontend logs 422/400 on label and metadata APIs
	// (see grafana/mimir#8990, grafana/mimir#8253).
	cfg := pulumi.Sprintf(`multitenancy_enabled: false
target: all

ingester:
  ring:
    replication_factor: 1

store_gateway:
  sharding_ring:
    replication_factor: 1

common:
  storage:
    backend: s3
    s3:
      endpoint: %s
      access_key_id: ${S3_ACCESS_KEY}
      secret_access_key: ${S3_SECRET_KEY}
      insecure: %t
      bucket_name: %s

blocks_storage:
  storage_prefix: blocks
  tsdb:
    dir: /data/ingester

ruler:
  rule_path: /data/ruler

alertmanager:
  data_dir: /data/alertmanager
`,
		a.ObjectStorage.S3.Addr(),
		a.ObjectStorage.Insecure,
		pulumi.String(s3Bucket),
	)

	cmOpts := []pulumi.ResourceOption{pulumi.Parent(parent)}
	if d := a.ObjectStorage.Deps; len(d) > 0 {
		cmOpts = append(cmOpts, pulumi.DependsOn(d))
	}

	cm, err := corev1.NewConfigMap(ctx, a.Name+"-config", &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(a.Name + "-config"),
			Namespace: pulumi.String(a.Namespace),
		},
		Data: pulumi.StringMap{
			"mimir.yaml": cfg,
		},
	}, cmOpts...)
	if err != nil {
		return nil, fmt.Errorf("configmap: %w", err)
	}

	dep, err := appsv1.NewDeployment(ctx, a.Name, &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(a.Name),
			Namespace: pulumi.String(a.Namespace),
			Labels:    pulumi.StringMap{"app": pulumi.String(a.Name)},
			Annotations: pulumi.StringMap{
				annotations.WaitForKey: pulumi.String(annotations.WaitForValueImmediate),
			},
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{"app": pulumi.String(a.Name)},
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: pulumi.StringMap{"app": pulumi.String(a.Name)},
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:  pulumi.String("mimir"),
							Image: pulumi.String(image),
							Args: pulumi.StringArray{
								pulumi.String("-config.file=/etc/mimir/mimir.yaml"),
								pulumi.String("-config.expand-env=true"),
							},
							Ports: corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{
									Name:          pulumi.String("http"),
									ContainerPort: pulumi.Int(8080),
								},
							},
							Env: corev1.EnvVarArray{
								&corev1.EnvVarArgs{Name: pulumi.String("S3_ACCESS_KEY"), Value: a.ObjectStorage.Creds.AccessKey},
								&corev1.EnvVarArgs{Name: pulumi.String("S3_SECRET_KEY"), Value: a.ObjectStorage.Creds.SecretKey},
							},
							VolumeMounts: corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("config"),
									MountPath: pulumi.String("/etc/mimir"),
								},
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("data"),
									MountPath: pulumi.String("/data"),
								},
							},
						},
					},
					Volumes: corev1.VolumeArray{
						&corev1.VolumeArgs{
							Name: pulumi.String("config"),
							ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
								Name: cm.Metadata.Name().Elem(),
								Items: corev1.KeyToPathArray{
									&corev1.KeyToPathArgs{
										Key:  pulumi.String("mimir.yaml"),
										Path: pulumi.String("mimir.yaml"),
									},
								},
							},
						},
						&corev1.VolumeArgs{
							Name: pulumi.String("data"),
							EmptyDir: &corev1.EmptyDirVolumeSourceArgs{
								Medium: pulumi.String(""),
							},
						},
					},
				},
			},
		},
	}, pulumi.Parent(parent), pulumi.DependsOn([]pulumi.Resource{cm}))
	if err != nil {
		return nil, fmt.Errorf("deployment: %w", err)
	}

	svc, err := corev1.NewService(ctx, a.Name, &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(a.Name),
			Namespace: pulumi.String(a.Namespace),
		},
		Spec: &corev1.ServiceSpecArgs{
			Selector: pulumi.StringMap{"app": pulumi.String(a.Name)},
			Ports: corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Name:       pulumi.String("http"),
					Port:       pulumi.Int(80),
					TargetPort: pulumi.Int(8080),
				},
			},
		},
	}, pulumi.Parent(parent), pulumi.DependsOn([]pulumi.Resource{dep}))
	if err != nil {
		return nil, fmt.Errorf("service: %w", err)
	}

	host := fmt.Sprintf("%s.%s.svc.cluster.local", a.Name, a.Namespace)
	return &Result{
		Gateway:    endpoint.HTTP(host, "80", "/"),
		Prometheus: endpoint.HTTP(host, "80", "/prometheus"),
		Push:       endpoint.HTTP(host, "80", "/api/v1/push"),
		Deployment: dep,
		Service:    svc,
	}, nil
}
