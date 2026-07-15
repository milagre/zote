// Package body is the shared Deployment; labels service=<Type>, name=<Name>, app=<Name>, deploy=<Type>.
package body

import (
	"fmt"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/k8s/deployment/internal/scaledobject"
	"github.com/milagre/zote/pulumi/k8s/internal/podspec"
	"github.com/milagre/zote/pulumi/util/annotations"
	"github.com/milagre/zote/pulumi/util/labels"
	"github.com/milagre/zote/pulumi/util/profile"
)

const metricsPortName = "metrics"

type Args struct {
	Env       env.Env
	Name      string
	Namespace string
	Type      string // deploy label; narrows Service selectors vs same app name

	Image   string
	Tag     string
	Command []string
	Args    []string
	Profile profile.Profile

	Conf              podspec.Conf
	Files             podspec.Files
	Ports             []podspec.Port
	HTTPLivenessProbe *podspec.HTTPLivenessProbe

	// Metrics=true enables scraping of the container's /metrics endpoint.
	Metrics bool

	// Autoscale, when set, emits a KEDA ScaledObject for this Deployment.
	// Requires Cluster.HasKeda. When present, Pulumi stops managing
	// spec.replicas so it does not fight the autoscaler.
	Autoscale *scaledobject.Spec

	// Cluster supplies autoscaling coordinates (KEDA availability, metrics
	// query endpoint). Required when Autoscale is set.
	Cluster *infra.Cluster
}

// Register creates the Deployment.
func Register(
	ctx *pulumi.Context,
	name string,
	args Args,
	parent pulumi.Resource,
	opts ...pulumi.ResourceOption,
) (*appsv1.Deployment, error) {
	if args.Name == "" {
		return nil, fmt.Errorf("body: Name is required")
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("body: Namespace is required")
	}
	if args.Type == "" {
		return nil, fmt.Errorf("body: Type is required")
	}

	podLabels := labels.Pod(args.Namespace, args.Name)
	podLabels["app"] = pulumi.String(args.Name)
	podLabels["deploy"] = pulumi.String(args.Type)

	podAnnotations := pulumi.StringMap{}
	if args.Metrics {
		for k, v := range metricsAnnotations(MetricsListenPort) {
			podAnnotations[k] = pulumi.String(v)
		}
	}

	if args.Profile.Num == nil {
		return nil, fmt.Errorf("body: Profile.Num is required for a Deployment")
	}

	if err := validateAutoscale(args); err != nil {
		return nil, fmt.Errorf("body: %w", err)
	}

	spec, err := podspec.Build(podspec.Args{
		Env:               args.Env,
		Name:              args.Name,
		Namespace:         args.Namespace,
		Image:             args.Image,
		Tag:               args.Tag,
		ImagePullPolicy:   "IfNotPresent",
		Command:           args.Command,
		Args:              args.Args,
		Profile:           args.Profile,
		Conf:              args.Conf,
		Files:             args.Files,
		Ports:             args.Ports,
		HTTPLivenessProbe: args.HTTPLivenessProbe,
	})
	if err != nil {
		return nil, fmt.Errorf("body: %w", err)
	}

	childOpts := append([]pulumi.ResourceOption{pulumi.Parent(parent)}, opts...)

	ns := pulumi.String(args.Namespace)

	// When KEDA owns this Deployment's replica count, Pulumi must stop
	// reconciling spec.replicas or the two controllers fight on every apply.
	var ignoredFields []string
	if args.Autoscale != nil {
		ignoredFields = []string{"spec.replicas"}
	}

	dep, err := appsv1.NewDeployment(ctx, name, &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(args.Name),
			Namespace: ns,
			Labels:    podLabels,
			Annotations: pulumi.StringMap{
				annotations.WaitForKey: pulumi.String(annotations.WaitForValueImmediate),
			},
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(args.Profile.Num.Min),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: podLabels,
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:        pulumi.String(args.Name),
					Namespace:   ns,
					Labels:      podLabels,
					Annotations: podAnnotations,
				},
				Spec: spec,
			},
		},
	},
		append(childOpts, pulumi.IgnoreChanges(ignoredFields))...,
	)
	if err != nil {
		return nil, fmt.Errorf("body: deployment: %w", err)
	}

	if args.Autoscale != nil {
		if err := registerAutoscaler(ctx, name, args, dep); err != nil {
			return nil, fmt.Errorf("body: %w", err)
		}
	}

	return dep, nil
}

// validateAutoscale enforces the invariants the autoscaler depends on before
// any resources are created: KEDA must be installed, HTTP workloads must keep a
// floor of at least one replica (something must always serve requests), and a
// utilization trigger needs the metrics store to be resolvable.
func validateAutoscale(args Args) error {
	if args.Autoscale == nil {
		return nil
	}
	if args.Cluster == nil || !args.Cluster.HasKeda {
		return fmt.Errorf("autoscale requires KEDA on the cluster")
	}
	if args.Type == "http" && args.Profile.Num.Min < 1 {
		return fmt.Errorf("autoscale: http workloads require Profile.Num.Min >= 1 (cannot scale to zero)")
	}

	// The metrics endpoint is resolved from the cluster only when the caller
	// did not pin its own ServerAddress.
	if u := args.Autoscale.Utilization; u != nil && u.ServerAddress == "" && args.Cluster.MetricsQueryURL == nil {
		return fmt.Errorf("autoscale: utilization trigger needs a metrics query endpoint (set ServerAddress or register the stats stack)")
	}

	return nil
}

// registerAutoscaler builds the ScaledObject for dep, resolving the utilization
// trigger's metrics endpoint from the cluster when the caller left it unset. The
// metric query itself is caller-supplied (see deployment.ZAMQPUtilizationStat
// for the zamqp-consumer case) because not every proc workload emits the same
// utilization signal.
func registerAutoscaler(ctx *pulumi.Context, name string, args Args, dep pulumi.Resource) error {
	spec := *args.Autoscale
	if u := spec.Utilization; u != nil {
		util := *u
		if util.ServerAddress == "" {
			util.ServerAddress = *args.Cluster.MetricsQueryURL
		}
		spec.Utilization = &util
	}

	if err := scaledobject.Register(ctx, name+"-autoscale", scaledobject.Args{
		Namespace:  args.Namespace,
		TargetName: args.Name,
		Min:        args.Profile.Num.Min,
		Max:        args.Profile.Num.Max,
		Spec:       spec,
	}, dep); err != nil {
		return fmt.Errorf("autoscale: %w", err)
	}

	return nil
}
