package container

import (
	"strconv"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	portPrometheus = 9121
	exporterImage  = "quay.io/oliver006/redis_exporter:v1.90.0"
)

// PrometheusPodAnnotations are pod template annotations for Alloy's annotation-based scrape.
func PrometheusPodAnnotations() map[string]string {
	return map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/path":   "/metrics",
		"prometheus.io/port":   strconv.Itoa(portPrometheus),
	}
}

func prometheusPodAnnotationMap() pulumi.StringMap {
	out := pulumi.StringMap{}
	for k, v := range PrometheusPodAnnotations() {
		out[k] = pulumi.String(v)
	}

	return out
}

func redisExporterContainer() *corev1.ContainerArgs {
	return &corev1.ContainerArgs{
		Name:            pulumi.String("redis-exporter"),
		Image:           pulumi.String(exporterImage),
		ImagePullPolicy: pulumi.String("IfNotPresent"),
		Args: pulumi.StringArray{
			pulumi.String("--redis.addr=redis://127.0.0.1:6379"),
		},
		Ports: corev1.ContainerPortArray{
			&corev1.ContainerPortArgs{Name: pulumi.String("metrics"), ContainerPort: pulumi.Int(portPrometheus), Protocol: pulumi.String("TCP")},
		},
		Resources: &corev1.ResourceRequirementsArgs{
			Limits: pulumi.StringMap{
				"cpu":    pulumi.String("50m"),
				"memory": pulumi.String("64Mi"),
			},
			Requests: pulumi.StringMap{
				"cpu":    pulumi.String("10m"),
				"memory": pulumi.String("16Mi"),
			},
		},
	}
}
