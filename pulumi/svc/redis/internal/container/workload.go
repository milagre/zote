package container

import (
	"fmt"
	"math"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/util/annotations"
	"github.com/milagre/zote/pulumi/util/labels"
)

const scriptsDefaultMode = 0o755

func registerStatefulSet(
	ctx *pulumi.Context,
	parentName string,
	comp pulumi.Resource,
	args *Args,
	releaseName string,
	svc *corev1.Service,
	cfg *corev1.ConfigMap,
	scripts *corev1.ConfigMap,
) (*appsv1.StatefulSet, error) {
	if args.Standard {
		return registerStandardStatefulSet(ctx, parentName, comp, args, releaseName, svc, cfg)
	}
	if scripts == nil {
		return nil, fmt.Errorf("cluster mode requires scripts ConfigMap")
	}

	return registerClusterStatefulSet(ctx, parentName, comp, args, releaseName, svc, cfg, scripts)
}

func registerStandardStatefulSet(
	ctx *pulumi.Context,
	parentName string,
	comp pulumi.Resource,
	args *Args,
	releaseName string,
	svc *corev1.Service,
	cfg *corev1.ConfigMap,
) (*appsv1.StatefulSet, error) {
	podLabels := labels.Pod("redis", args.Name)
	storageMB := int64(math.Floor(float64(args.Profile.MemMB.Max) * 1.1))

	return appsv1.NewStatefulSet(ctx, parentName, &appsv1.StatefulSetArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(releaseName),
			Namespace: pulumi.String(args.Namespace),
			Annotations: pulumi.StringMap{
				annotations.WaitForKey: pulumi.String(annotations.WaitForValueImmediate),
			},
		},
		Spec: &appsv1.StatefulSetSpecArgs{
			Replicas:            pulumi.Int(1),
			ServiceName:         svc.Metadata.Name().Elem(),
			PodManagementPolicy: pulumi.String("OrderedReady"),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: podLabels,
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels:      podLabels,
					Annotations: prometheusPodAnnotationMap(),
				},
				Spec: &corev1.PodSpecArgs{
					TerminationGracePeriodSeconds: pulumi.Int(60),
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:            pulumi.String("redis"),
							Image:           pulumi.String(fmt.Sprintf("redis:%s", args.Version)),
							ImagePullPolicy: pulumi.String("IfNotPresent"),
							Command: pulumi.StringArray{
								pulumi.String("redis-server"),
								pulumi.String("/etc/redis/redis.conf"),
							},
							Lifecycle: &corev1.LifecycleArgs{
								PreStop: &corev1.LifecycleHandlerArgs{
									Exec: &corev1.ExecActionArgs{
										Command: pulumi.StringArray{
											pulumi.String("sh"),
											pulumi.String("-c"),
											pulumi.String("redis-cli -p 6379 SHUTDOWN 2>/dev/null || true"),
										},
									},
								},
							},
							Ports: corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{Name: pulumi.String("client"), ContainerPort: pulumi.Int(clientPort)},
							},
							VolumeMounts: corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{Name: pulumi.String("data"), MountPath: pulumi.String("/data")},
								&corev1.VolumeMountArgs{Name: pulumi.String("config"), MountPath: pulumi.String("/etc/redis")},
							},
							Resources: &corev1.ResourceRequirementsArgs{
								Limits: pulumi.StringMap{
									"cpu":    pulumi.Sprintf("%g", args.Profile.CPUCores.Max),
									"memory": pulumi.Sprintf("%dM", args.Profile.MemMB.Max),
								},
								Requests: pulumi.StringMap{
									"cpu":    pulumi.Sprintf("%g", args.Profile.CPUCores.Min),
									"memory": pulumi.Sprintf("%dM", args.Profile.MemMB.Min),
								},
							},
						},
						redisExporterContainer(),
					},
					Volumes: corev1.VolumeArray{
						&corev1.VolumeArgs{
							Name: pulumi.String("config"),
							ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
								Name: cfg.Metadata.Name().Elem(),
							},
						},
					},
				},
			},
			VolumeClaimTemplates: corev1.PersistentVolumeClaimTypeArray{
				&corev1.PersistentVolumeClaimTypeArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Name: pulumi.String("data"),
					},
					Spec: &corev1.PersistentVolumeClaimSpecArgs{
						AccessModes: pulumi.StringArray{pulumi.String("ReadWriteOnce")},
						Resources: &corev1.VolumeResourceRequirementsArgs{
							Requests: pulumi.StringMap{
								"storage": pulumi.Sprintf("%dM", storageMB),
							},
						},
					},
				},
			},
		},
	},
		pulumi.Parent(comp),
		pulumi.IgnoreChanges([]string{
			"spec.volumeClaimTemplates",
		}),
	)
}

func registerClusterStatefulSet(
	ctx *pulumi.Context,
	parentName string,
	comp pulumi.Resource,
	args *Args,
	releaseName string,
	svc *corev1.Service,
	cfg *corev1.ConfigMap,
	scripts *corev1.ConfigMap,
) (*appsv1.StatefulSet, error) {
	podLabels := labels.Pod("redis", args.Name)
	totalPods := (args.Replicas + 1) * args.Shards
	storageMB := int64(math.Floor(float64(args.Profile.MemMB.Max) * 1.1))

	return appsv1.NewStatefulSet(ctx, parentName, &appsv1.StatefulSetArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(releaseName),
			Namespace: pulumi.String(args.Namespace),
			Annotations: pulumi.StringMap{
				annotations.WaitForKey: pulumi.String(annotations.WaitForValueImmediate),
			},
		},
		Spec: &appsv1.StatefulSetSpecArgs{
			Replicas:            pulumi.Int(totalPods),
			ServiceName:         svc.Metadata.Name().Elem(),
			PodManagementPolicy: pulumi.String("OrderedReady"),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: podLabels,
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels:      podLabels,
					Annotations: prometheusPodAnnotationMap(),
				},
				Spec: &corev1.PodSpecArgs{
					TerminationGracePeriodSeconds: pulumi.Int(60),
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:            pulumi.String("redis"),
							Image:           pulumi.String(fmt.Sprintf("redis:%s", args.Version)),
							ImagePullPolicy: pulumi.String("IfNotPresent"),
							Command: pulumi.StringArray{
								pulumi.String("/etc/scripts/update-nodes.sh"),
								pulumi.String("redis-server"),
								pulumi.String("/etc/redis/redis.conf"),
							},
							Lifecycle: &corev1.LifecycleArgs{
								PreStop: &corev1.LifecycleHandlerArgs{
									Exec: &corev1.ExecActionArgs{
										Command: pulumi.StringArray{
											pulumi.String("sh"),
											pulumi.String("-c"),
											pulumi.String("redis-cli -p 6379 SHUTDOWN 2>/dev/null || true"),
										},
									},
								},
							},
							Ports: corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{Name: pulumi.String("client"), ContainerPort: pulumi.Int(clientPort)},
								&corev1.ContainerPortArgs{Name: pulumi.String("cluster"), ContainerPort: pulumi.Int(clusterPort)},
							},
							Env: corev1.EnvVarArray{
								&corev1.EnvVarArgs{
									Name: pulumi.String("POD_IP"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										FieldRef: &corev1.ObjectFieldSelectorArgs{FieldPath: pulumi.String("status.podIP")},
									},
								},
								&corev1.EnvVarArgs{
									Name: pulumi.String("POD_NAME"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										FieldRef: &corev1.ObjectFieldSelectorArgs{FieldPath: pulumi.String("metadata.name")},
									},
								},
								&corev1.EnvVarArgs{
									Name: pulumi.String("POD_NAMESPACE"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										FieldRef: &corev1.ObjectFieldSelectorArgs{FieldPath: pulumi.String("metadata.namespace")},
									},
								},
								&corev1.EnvVarArgs{
									Name:  pulumi.String("REDIS_HEADLESS_SERVICE"),
									Value: svc.Metadata.Name().Elem(),
								},
							},
							VolumeMounts: corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{Name: pulumi.String("data"), MountPath: pulumi.String("/data")},
								&corev1.VolumeMountArgs{Name: pulumi.String("config"), MountPath: pulumi.String("/etc/redis")},
								&corev1.VolumeMountArgs{Name: pulumi.String("scripts"), MountPath: pulumi.String("/etc/scripts")},
							},
							Resources: &corev1.ResourceRequirementsArgs{
								Limits: pulumi.StringMap{
									"cpu":    pulumi.Sprintf("%g", args.Profile.CPUCores.Max),
									"memory": pulumi.Sprintf("%dM", args.Profile.MemMB.Max),
								},
								Requests: pulumi.StringMap{
									"cpu":    pulumi.Sprintf("%g", args.Profile.CPUCores.Min),
									"memory": pulumi.Sprintf("%dM", args.Profile.MemMB.Min),
								},
							},
						},
						redisExporterContainer(),
					},
					Volumes: corev1.VolumeArray{
						&corev1.VolumeArgs{
							Name: pulumi.String("config"),
							ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
								Name: cfg.Metadata.Name().Elem(),
							},
						},
						&corev1.VolumeArgs{
							Name: pulumi.String("scripts"),
							ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
								Name:        scripts.Metadata.Name().Elem(),
								DefaultMode: pulumi.Int(scriptsDefaultMode),
							},
						},
						&corev1.VolumeArgs{
							Name:     pulumi.String("shared"),
							EmptyDir: &corev1.EmptyDirVolumeSourceArgs{},
						},
					},
				},
			},
			VolumeClaimTemplates: corev1.PersistentVolumeClaimTypeArray{
				&corev1.PersistentVolumeClaimTypeArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Name: pulumi.String("data"),
					},
					Spec: &corev1.PersistentVolumeClaimSpecArgs{
						AccessModes: pulumi.StringArray{pulumi.String("ReadWriteOnce")},
						Resources: &corev1.VolumeResourceRequirementsArgs{
							Requests: pulumi.StringMap{
								"storage": pulumi.Sprintf("%dM", storageMB),
							},
						},
					},
				},
			},
		},
	},
		pulumi.Parent(comp),
		pulumi.IgnoreChanges([]string{
			// VolumeClaimTemplates is full of server-populated metadata/status that
			// pulumi-kubernetes reports as diffs and forces StatefulSet replacement on.
			"spec.volumeClaimTemplates",
		}),
	)
}
