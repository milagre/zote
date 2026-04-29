// Package body is the shared Deployment + PodMonitor assembly that the
// http and proc workload wrappers both rely on.
//
// The body owns the Kubernetes Deployment resource and its PodSpec, the
// label convention (`app=<Name>`, `deploy=<Type>`), and the adjacent
// PodMonitor that is emitted automatically when a port named "metrics"
// is declared. Wrappers attach a Service/Ingress layer on top and never
// reach into the Deployment's internals.
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

// Args is the input set for the body. Type selects the value of the
// `deploy` label (e.g. "http", "proc"); it does not alter the resources
// that are created, but downstream Services use it to narrow their
// selectors so that a namespace can host multiple workloads with
// distinct deploy types on the same app name.
type Args struct {
	Env       env.Env
	Name      string
	Namespace string
	Type      string

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

// Register creates the Deployment (and, when a "metrics" port is
// declared, the companion PodMonitor) as children of parent. The
// Deployment is returned so wrappers can express explicit dependencies
// on it (e.g. ordering a Service creation after the Deployment); no
// fields of the returned resource are part of the wrappers' public API.
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
			Annotations: annotations.Managed(),
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
			Annotations: annotations.Managed(),
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
