// Package body is the shared Deployment (+ PodMonitor when a "metrics" port exists); labels app=<Name>, deploy=<Type>.
package body

import (
	"fmt"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/k8s/internal/annotations"
	"github.com/milagre/zote/pulumi/k8s/internal/podspec"
	"github.com/milagre/zote/pulumi/profile"
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
}

// Register creates the Deployment (and PodMonitor if a "metrics" port exists).
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

	labels := pulumi.StringMap{
		"app":    pulumi.String(args.Name),
		"deploy": pulumi.String(args.Type),
	}

	if args.Profile.Num == nil {
		return nil, fmt.Errorf("body: Profile.Num is required for a Deployment")
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
	dep, err := appsv1.NewDeployment(ctx, name, &appsv1.DeploymentArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(args.Name),
			Namespace:   ns,
			Labels:      labels,
			Annotations: pulumi.StringMap{
				annotations.SkipAwaitKey: pulumi.String(annotations.SkipAwaitValueReady),
			},
		},
		Spec: &appsv1.DeploymentSpecArgs{
			Replicas: pulumi.Int(args.Profile.Num.Min),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: labels,
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String(args.Name),
					Namespace: ns,
					Labels:    labels,
				},
				Spec: spec,
			},
		},
	},
		append(childOpts, pulumi.IgnoreChanges([]string{
			// Targeting a specific key (e.g. annotations["kubectl.kubernetes.io/restartedAt"])
			// hits "cannot ignore changes in added or removed elements" when the key is
			// only present on one side of the diff; widening to the containing map works.
			"spec.template.metadata.annotations",
		}))...,
	)
	if err != nil {
		return nil, fmt.Errorf("body: deployment: %w", err)
	}

	if hasMetricsPort(args.Ports) {
		if _, err := registerPodMonitor(ctx, name, args, labels, parent, opts); err != nil {
			return nil, err
		}
	}

	return dep, nil
}

func hasMetricsPort(ports []podspec.Port) bool {
	for _, p := range ports {
		if p.Name == metricsPortName {
			return true
		}
	}

	return false
}

// registerPodMonitor emits the prometheus-operator PodMonitor CR that
// scrapes the Deployment's metrics port. It is created via the generic
// CustomResource factory so the Zote library does not need a generated
// binding for the prometheus-operator CRDs.
func registerPodMonitor(
	ctx *pulumi.Context,
	name string,
	args Args,
	labels pulumi.StringMap,
	parent pulumi.Resource,
	opts []pulumi.ResourceOption,
) (*apiextensions.CustomResource, error) {
	childOpts := append([]pulumi.ResourceOption{pulumi.Parent(parent)}, opts...)

	cr, err := apiextensions.NewCustomResource(ctx, name+"-podmonitor", &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("monitoring.coreos.com/v1"),
		Kind:       pulumi.String("PodMonitor"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(args.Name),
			Namespace: pulumi.String(args.Namespace),
			Labels: pulumi.StringMap{
				"app": pulumi.String(args.Name),
			},
			Annotations: pulumi.StringMap{
				annotations.SkipAwaitKey: pulumi.String(annotations.SkipAwaitValueReady),
			},
		},
		OtherFields: kubernetes.UntypedArgs{
			"spec": pulumi.Map{
				"selector": pulumi.Map{
					"matchLabels": labels,
				},
				"namespaceSelector": pulumi.Map{
					"matchNames": pulumi.StringArray{pulumi.String(args.Namespace)},
				},
				"podMetricsEndpoints": pulumi.Array{
					pulumi.Map{
						"port": pulumi.String(metricsPortName),
						"path": pulumi.String("/metrics"),
					},
				},
			},
		},
	}, childOpts...)
	if err != nil {
		return nil, fmt.Errorf("body: podmonitor: %w", err)
	}

	return cr, nil
}
