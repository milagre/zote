package container

import (
	"fmt"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	batchv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/batch/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func registerBootstrapJob(
	ctx *pulumi.Context,
	parentName string,
	comp pulumi.Resource,
	args *Args,
	releaseName string,
	scripts *corev1.ConfigMap,
	sts *appsv1.StatefulSet,
) (*batchv1.Job, error) {
	totalPods := (args.Replicas + 1) * args.Shards

	return batchv1.NewJob(ctx, parentName+"-cluster", &batchv1.JobArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(releaseName + "-cluster"),
			Namespace: pulumi.String(args.Namespace),
		},
		Spec: &batchv1.JobSpecArgs{
			BackoffLimit: pulumi.Int(20),
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name: pulumi.String(releaseName + "-cluster"),
				},
				Spec: &corev1.PodSpecArgs{
					RestartPolicy: pulumi.String("Never"),
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:            pulumi.String("cluster-bootstrap"),
							Image:           pulumi.String(fmt.Sprintf("redis:%s", args.Version)),
							ImagePullPolicy: pulumi.String("IfNotPresent"),
							Command:         pulumi.StringArray{pulumi.String("/etc/scripts/cluster-bootstrap.sh")},
							Env: corev1.EnvVarArray{
								&corev1.EnvVarArgs{Name: pulumi.String("REDIS_HEADLESS_NAME"), Value: pulumi.String(releaseName)},
								&corev1.EnvVarArgs{Name: pulumi.String("REDIS_CLUSTER_REPLICAS"), Value: pulumi.Sprintf("%d", args.Replicas)},
								&corev1.EnvVarArgs{Name: pulumi.String("REDIS_NODE_COUNT"), Value: pulumi.Sprintf("%d", totalPods)},
								&corev1.EnvVarArgs{Name: pulumi.String("REDIS_CLUSTER_PORT"), Value: pulumi.Sprintf("%d", clientPort)},
							},
							VolumeMounts: corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("scripts"),
									MountPath: pulumi.String("/etc/scripts"),
								},
							},
						},
					},
					Volumes: corev1.VolumeArray{
						&corev1.VolumeArgs{
							Name: pulumi.String("scripts"),
							ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
								Name:        scripts.Metadata.Name().Elem(),
								DefaultMode: pulumi.Int(scriptsDefaultMode),
							},
						},
					},
				},
			},
		},
	},
		pulumi.Parent(comp),
		pulumi.DependsOn([]pulumi.Resource{sts}),
	)
}
