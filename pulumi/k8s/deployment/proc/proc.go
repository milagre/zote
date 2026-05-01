// Package proc is a non-HTTP Deployment plus headless ClusterIP for per-pod DNS.
package proc

import (
	"fmt"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/k8s/deployment/internal/body"
	"github.com/milagre/zote/pulumi/k8s/internal/annotations"
	"github.com/milagre/zote/pulumi/k8s/internal/podspec"
	"github.com/milagre/zote/pulumi/profile"
	"github.com/milagre/zote/pulumi/tokens"
)

var typeToken = tokens.Token("k8s", "ProcDeployment")

type Conf = podspec.Conf
type Files = podspec.Files

type Args struct {
	Env       env.Env
	Namespace string
	Name      string

	Image   string
	Tag     string
	Command []string
	Args    []string
	Profile profile.Profile

	Conf  Conf
	Files Files

	PrometheusMonitored bool
}

type Deployment struct {
	pulumi.ResourceState
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Deployment, error) {
	if args == nil {
		return nil, fmt.Errorf("%s: args is required", typeToken)
	}
	if err := args.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	resourceName := tokens.Qualify(args.Namespace, name)

	comp := &Deployment{}
	if err := ctx.RegisterComponentResource(typeToken, resourceName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	var ports []podspec.Port
	if args.PrometheusMonitored {
		ports = append(ports, podspec.Port{
			Name:          "metrics",
			ContainerPort: 9090,
			Protocol:      "TCP",
		})
	}

	if _, err := body.Register(ctx, resourceName, body.Args{
		Env:       args.Env,
		Name:      args.Name,
		Namespace: args.Namespace,
		Type:      "proc",
		Image:     args.Image,
		Tag:       args.Tag,
		Command:   args.Command,
		Args:      args.Args,
		Profile:   args.Profile,
		Conf:      args.Conf,
		Files:     args.Files,
		Ports:     ports,
	}, comp); err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	if _, err := registerService(ctx, resourceName, args, comp); err != nil {
		return nil, err
	}

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

func (a *Args) validate() error {
	if a.Name == "" {
		return fmt.Errorf("Name is required")
	}
	if a.Namespace == "" {
		return fmt.Errorf("Namespace is required")
	}
	if a.Image == "" {
		return fmt.Errorf("Image is required")
	}
	if a.Tag == "" {
		return fmt.Errorf("Tag is required")
	}
	if err := a.Env.Validate(); err != nil {
		return err
	}

	return nil
}

// registerService creates a headless (ClusterIP "None") Service scoped
// to this workload. The service carries a single port so pod DNS
// entries have a well-formed endpoint; proc workloads don't actually
// serve HTTP, but a headless service lets dependents discover pod IPs
// through the cluster DNS without a separate mechanism.
func registerService(ctx *pulumi.Context, name string, args *Args, parent pulumi.Resource) (*corev1.Service, error) {
	svc, err := corev1.NewService(ctx, name, &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(args.Name),
			Namespace:   pulumi.String(args.Namespace),
			Annotations: annotations.Managed(),
		},
		Spec: &corev1.ServiceSpecArgs{
			ClusterIP: pulumi.String("None"),
			Selector: pulumi.StringMap{
				"app":    pulumi.String(args.Name),
				"deploy": pulumi.String("proc"),
			},
			Ports: corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Port:       pulumi.Int(80),
					TargetPort: pulumi.Int(80),
					Protocol:   pulumi.String("TCP"),
				},
			},
		},
	}, pulumi.Parent(parent))
	if err != nil {
		return nil, fmt.Errorf("service: %w", err)
	}

	return svc, nil
}
