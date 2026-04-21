package container

import (
	"fmt"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func registerHeadlessService(
	ctx *pulumi.Context,
	parentName string,
	comp pulumi.Resource,
	namespace string,
	releaseName string,
) (*corev1.Service, error) {
	labels := pulumi.StringMap{"app": pulumi.String(releaseName)}
	// Headless: required for StatefulSet per-pod DNS (redis-0.redis-name.ns.svc)
	// so cluster hostnames and CLUSTER MEET targets resolve to current Pod IPs
	// after restarts.
	return corev1.NewService(ctx, parentName, &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(releaseName),
			Namespace: pulumi.String(namespace),
			Labels:    labels,
		},
		Spec: &corev1.ServiceSpecArgs{
			ClusterIP: pulumi.String("None"),
			Selector:  labels,
			Ports: corev1.ServicePortArray{
				&corev1.ServicePortArgs{Port: pulumi.Int(clientPort)},
			},
		},
	}, pulumi.Parent(comp))
}

func registerConfig(
	ctx *pulumi.Context,
	parentName string,
	comp pulumi.Resource,
	namespace string,
	releaseName string,
) (*corev1.ConfigMap, *corev1.ConfigMap, error) {
	labels := pulumi.StringMap{"app": pulumi.String(releaseName)}
	ns := pulumi.String(namespace)
	patchForce := pulumi.StringMap{"pulumi.com/patchForce": pulumi.String("true")}

	cfg, err := corev1.NewConfigMap(ctx, parentName+"-conf", &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String("cfg-" + releaseName),
			Namespace:   ns,
			Labels:      labels,
			Annotations: patchForce,
		},
		Data: pulumi.StringMap{
			"redis.conf": pulumi.String(redisConf),
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, nil, fmt.Errorf("redis.conf configmap: %w", err)
	}

	scripts, err := corev1.NewConfigMap(ctx, parentName+"-scripts", &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String("cfg-" + releaseName + "-scripts"),
			Namespace:   ns,
			Labels:      labels,
			Annotations: patchForce,
		},
		Data: pulumi.StringMap{
			"update-nodes.sh":      pulumi.String(updateNodesScript),
			"cluster-bootstrap.sh": pulumi.String(clusterBootstrapScript),
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, nil, fmt.Errorf("scripts configmap: %w", err)
	}

	return cfg, scripts, nil
}
