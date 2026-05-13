// Package body is the shared Deployment; labels app=<Name>, deploy=<Type>.
package body

import (
	"fmt"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/annotations"
	"github.com/milagre/zote/pulumi/env"
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

	// Metrics=true enables scraping of the container's /metrics endpoint.
	Metrics bool
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

	podLabels := pulumi.StringMap{
		"app":    pulumi.String(args.Name),
		"deploy": pulumi.String(args.Type),
	}

	podAnnotations := pulumi.StringMap{}
	if args.Metrics {
		for k, v := range metricsAnnotations(MetricsListenPort) {
			podAnnotations[k] = pulumi.String(v)
		}
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
			Name:      pulumi.String(args.Name),
			Namespace: ns,
			Labels:    podLabels,
			Annotations: pulumi.StringMap{
				annotations.SkipAwaitKey: pulumi.String(annotations.SkipAwaitValueAll),
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
		append(childOpts, pulumi.IgnoreChanges([]string{}))...,
	)
	if err != nil {
		return nil, fmt.Errorf("body: deployment: %w", err)
	}

	return dep, nil
}
