package container

import (
	"fmt"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/profile"
)

const (
	portAMQP        = 5672
	portManagement  = 15672
	portPrometheus  = 15692
	portEPMD        = 4369
	portClusterRPC  = 25672
	dataStorageSize = "2Gi"
)

// registerWorkload creates the StatefulSet, client Service and headless
// Service that define the rabbitmq cluster.
func registerWorkload(
	ctx *pulumi.Context,
	parentName string,
	comp pulumi.Resource,
	namespace string,
	releaseName string,
	version string,
	prof profile.Profile,
	cfgCM *corev1.ConfigMap,
	cfgSecret *corev1.Secret,
	sa *corev1.ServiceAccount,
) (*appsv1.StatefulSet, *corev1.Service, *corev1.Service, error) {
	ns := pulumi.String(namespace)
	labels := pulumi.StringMap{"app": pulumi.String(releaseName)}
	replicas := 1
	if prof.Num != nil {
		replicas = prof.Num.Min
	}

	initCommand := []string{
		"sh", "-c",
		"cp /tmp/rabbitmq/rabbitmq.conf /etc/rabbitmq/rabbitmq.conf && echo '' >> /etc/rabbitmq/rabbitmq.conf; " +
			"cp /tmp/rabbitmq/enabled_plugins /etc/rabbitmq/enabled_plugins; " +
			"cp /tmp/rabbitmq/definitions.json /etc/rabbitmq/definitions.json",
	}

	initCmd := make(pulumi.StringArray, 0, len(initCommand))
	for _, a := range initCommand {
		initCmd = append(initCmd, pulumi.String(a))
	}

	sts, err := appsv1.NewStatefulSet(ctx, parentName, &appsv1.StatefulSetArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(releaseName),
			Namespace: ns,
		},
		Spec: &appsv1.StatefulSetSpecArgs{
			Replicas:            pulumi.Int(replicas),
			ServiceName:         pulumi.String(releaseName + "-headless"),
			PodManagementPolicy: pulumi.String("OrderedReady"),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: labels,
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:   pulumi.String(releaseName),
					Labels: labels,
				},
				Spec: &corev1.PodSpecArgs{
					ServiceAccountName: sa.Metadata.Name().Elem(),
					SecurityContext: &corev1.PodSecurityContextArgs{
						RunAsUser:  pulumi.Int(999),
						RunAsGroup: pulumi.Int(999),
						FsGroup:    pulumi.Int(999),
					},
					InitContainers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:    pulumi.String("rabbitmq-config"),
							Image:   pulumi.String("busybox:1.32.0"),
							Command: initCmd,
							VolumeMounts: corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{Name: pulumi.String("config"), MountPath: pulumi.String("/tmp/rabbitmq")},
								&corev1.VolumeMountArgs{Name: pulumi.String("config-rw"), MountPath: pulumi.String("/etc/rabbitmq")},
							},
						},
					},
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:  pulumi.String("rabbitmq"),
							Image: pulumi.String(fmt.Sprintf("rabbitmq:%s", version)),
							Ports: corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{Name: pulumi.String("amqp"), ContainerPort: pulumi.Int(portAMQP), Protocol: pulumi.String("TCP")},
								&corev1.ContainerPortArgs{Name: pulumi.String("management"), ContainerPort: pulumi.Int(portManagement), Protocol: pulumi.String("TCP")},
								&corev1.ContainerPortArgs{Name: pulumi.String("prometheus"), ContainerPort: pulumi.Int(portPrometheus), Protocol: pulumi.String("TCP")},
								&corev1.ContainerPortArgs{Name: pulumi.String("epmd"), ContainerPort: pulumi.Int(portEPMD), Protocol: pulumi.String("TCP")},
							},
							Env: corev1.EnvVarArray{
								&corev1.EnvVarArgs{
									Name: pulumi.String("RABBITMQ_DEFAULT_PASS"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										SecretKeyRef: &corev1.SecretKeySelectorArgs{
											Name: cfgSecret.Metadata.Name().Elem(),
											Key:  pulumi.String("password"),
										},
									},
								},
								&corev1.EnvVarArgs{
									Name: pulumi.String("RABBITMQ_DEFAULT_USER"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelectorArgs{
											Name: cfgCM.Metadata.Name().Elem(),
											Key:  pulumi.String("username"),
										},
									},
								},
								&corev1.EnvVarArgs{
									Name: pulumi.String("RABBITMQ_ERLANG_COOKIE"),
									ValueFrom: &corev1.EnvVarSourceArgs{
										SecretKeyRef: &corev1.SecretKeySelectorArgs{
											Name: cfgSecret.Metadata.Name().Elem(),
											Key:  pulumi.String("erlang_cookie"),
										},
									},
								},
							},
							VolumeMounts: corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{Name: pulumi.String("config-rw"), MountPath: pulumi.String("/etc/rabbitmq")},
								&corev1.VolumeMountArgs{Name: pulumi.String("data"), MountPath: pulumi.String("/var/lib/rabbitmq/mnesia")},
							},
							LivenessProbe: &corev1.ProbeArgs{
								Exec: &corev1.ExecActionArgs{
									Command: pulumi.StringArray{pulumi.String("rabbitmq-diagnostics"), pulumi.String("status")},
								},
								InitialDelaySeconds: pulumi.Int(60),
								TimeoutSeconds:      pulumi.Int(15),
								PeriodSeconds:       pulumi.Int(60),
							},
							ReadinessProbe: &corev1.ProbeArgs{
								Exec: &corev1.ExecActionArgs{
									Command: pulumi.StringArray{pulumi.String("rabbitmq-diagnostics"), pulumi.String("ping")},
								},
								InitialDelaySeconds: pulumi.Int(10),
								TimeoutSeconds:      pulumi.Int(10),
								PeriodSeconds:       pulumi.Int(10),
							},
							Resources: &corev1.ResourceRequirementsArgs{
								Requests: pulumi.StringMap{
									"cpu":    pulumi.Sprintf("%g", prof.CPUCores.Min),
									"memory": pulumi.Sprintf("%dM", prof.MemMB.Min),
								},
								Limits: pulumi.StringMap{
									"cpu":    pulumi.Sprintf("%g", prof.CPUCores.Max),
									"memory": pulumi.Sprintf("%dM", prof.MemMB.Max),
								},
							},
						},
					},
					Volumes: corev1.VolumeArray{
						&corev1.VolumeArgs{
							Name: pulumi.String("config"),
							ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
								Name: cfgCM.Metadata.Name().Elem(),
								Items: corev1.KeyToPathArray{
									&corev1.KeyToPathArgs{Key: pulumi.String("enabled_plugins"), Path: pulumi.String("enabled_plugins")},
									&corev1.KeyToPathArgs{Key: pulumi.String("rabbitmq.conf"), Path: pulumi.String("rabbitmq.conf")},
									&corev1.KeyToPathArgs{Key: pulumi.String("definitions.json"), Path: pulumi.String("definitions.json")},
								},
							},
						},
						&corev1.VolumeArgs{
							Name:     pulumi.String("config-rw"),
							EmptyDir: &corev1.EmptyDirVolumeSourceArgs{},
						},
						&corev1.VolumeArgs{
							Name: pulumi.String("data"),
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
								ClaimName: pulumi.String("data"),
							},
						},
					},
				},
			},
			VolumeClaimTemplates: corev1.PersistentVolumeClaimTypeArray{
				&corev1.PersistentVolumeClaimTypeArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Name:      pulumi.String("data"),
						Namespace: ns,
					},
					Spec: &corev1.PersistentVolumeClaimSpecArgs{
						AccessModes: pulumi.StringArray{pulumi.String("ReadWriteOnce")},
						Resources: &corev1.VolumeResourceRequirementsArgs{
							Requests: pulumi.StringMap{"storage": pulumi.String(dataStorageSize)},
						},
						StorageClassName: pulumi.String("standard"),
					},
				},
			},
		},
	},
		pulumi.Parent(comp),
		pulumi.DependsOn([]pulumi.Resource{cfgCM, cfgSecret}),
		pulumi.IgnoreChanges([]string{
			// VolumeClaimTemplates is full of server-populated metadata/status that
			// pulumi-kubernetes reports as diffs and forces StatefulSet replacement on.
			"spec.volumeClaimTemplates",
		}),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("statefulset: %w", err)
	}

	client, err := corev1.NewService(ctx, parentName, &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(releaseName),
			Namespace: ns,
			Labels:    labels,
		},
		Spec: &corev1.ServiceSpecArgs{
			Type:     pulumi.String("ClusterIP"),
			Selector: labels,
			Ports: corev1.ServicePortArray{
				&corev1.ServicePortArgs{Name: pulumi.String("http"), Protocol: pulumi.String("TCP"), Port: pulumi.Int(portManagement)},
				&corev1.ServicePortArgs{Name: pulumi.String("prometheus"), Protocol: pulumi.String("TCP"), Port: pulumi.Int(portPrometheus)},
				&corev1.ServicePortArgs{Name: pulumi.String("amqp"), Protocol: pulumi.String("TCP"), Port: pulumi.Int(portAMQP)},
			},
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("client service: %w", err)
	}

	headless, err := corev1.NewService(ctx, parentName+"-headless", &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(releaseName + "-headless"),
			Namespace: ns,
		},
		Spec: &corev1.ServiceSpecArgs{
			Type:            pulumi.String("ClusterIP"),
			ClusterIP:       pulumi.String("None"),
			SessionAffinity: pulumi.String("None"),
			Selector:        labels,
			Ports: corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Name:       pulumi.String("epmd"),
					Protocol:   pulumi.String("TCP"),
					Port:       pulumi.Int(portEPMD),
					TargetPort: pulumi.Int(portEPMD),
				},
				&corev1.ServicePortArgs{
					Name:       pulumi.String("cluster-rpc"),
					Protocol:   pulumi.String("TCP"),
					Port:       pulumi.Int(portClusterRPC),
					TargetPort: pulumi.Int(portClusterRPC),
				},
			},
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("headless service: %w", err)
	}

	return sts, client, headless, nil
}
