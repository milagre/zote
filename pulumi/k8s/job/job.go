// Package job provides a ComponentResource that runs a one-shot
// containerized task as a Kubernetes Job. The pod body is produced
// through the shared podspec builder so environment wiring, configmap/
// secret mounts, and the resource profile match every other workload in
// the library.
package job

import (
	"fmt"

	batchv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/batch/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/k8s/internal/annotations"
	"github.com/milagre/zote/pulumi/k8s/internal/podspec"
	"github.com/milagre/zote/pulumi/profile"
	"github.com/milagre/zote/pulumi/tokens"
)

var typeToken = tokens.Token("k8s", "Job")

// defaultBackoffLimit is the backoff limit used when the caller does
// not specify one. It is intentionally large because Jobs in this
// library are typically idempotent retries we expect to eventually
// succeed rather than fail-fast tasks.
const defaultBackoffLimit = 100000

// Conf is re-exported from the shared pod-spec layer.
type Conf = podspec.Conf

// Files mounts configmap data into the container filesystem.
type Files = podspec.Files

// Args configures a Job. Attempts, when nil, selects
// defaultBackoffLimit; setting it explicitly overrides that.
type Args struct {
	Env       env.Env
	Namespace string
	Name      string

	Image   string
	Tag     string
	Command []string
	Args    []string
	Profile profile.Profile

	Attempts *int
	Wait     bool

	Conf  Conf
	Files Files
}

// Job is the component resource. No outputs are exposed.
type Job struct {
	pulumi.ResourceState
}

// New registers a Job and its PodSpec as a single component.
func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Job, error) {
	if args == nil {
		return nil, fmt.Errorf("%s: args is required", typeToken)
	}
	if err := args.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	resourceName := tokens.Qualify(args.Namespace, name)

	comp := &Job{}
	if err := ctx.RegisterComponentResource(typeToken, resourceName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	spec, err := podspec.Build(podspec.Args{
		Env:             args.Env,
		Name:            args.Name,
		Namespace:       args.Namespace,
		Image:           args.Image,
		Tag:             args.Tag,
		ImagePullPolicy: imagePullPolicy(args.Env),
		Command:         args.Command,
		Args:            args.Args,
		Profile:         args.Profile,
		Conf:            args.Conf,
		Files:           args.Files,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	spec.RestartPolicy = pulumi.String("Never")

	backoff := defaultBackoffLimit
	if args.Attempts != nil {
		backoff = *args.Attempts
	}

	labels := pulumi.StringMap{"app": pulumi.String(args.Name)}

	var metaAnnotations pulumi.StringMap
	if !args.Wait {
		metaAnnotations = annotations.Managed()
	}

	if _, err := batchv1.NewJob(ctx, resourceName, &batchv1.JobArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(args.Name),
			Namespace:   pulumi.String(args.Namespace),
			Labels:      labels,
			Annotations: metaAnnotations,
		},
		Spec: &batchv1.JobSpecArgs{
			BackoffLimit: pulumi.Int(backoff),
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:   pulumi.String(args.Name),
					Labels: labels,
				},
				Spec: spec,
			},
		},
	}, pulumi.Parent(comp), pulumi.DeleteBeforeReplace(true)); err != nil {
		return nil, fmt.Errorf("%s: job: %w", typeToken, err)
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

// imagePullPolicy picks the container image pull policy based on the
// environment. Local clusters run against a local registry where no
// remote pull exists, so "Never" is safer than "IfNotPresent" (which
// would retry pulls after image deletions). Non-local clusters always
// fall back to a fresh pull when no cached image is present.
func imagePullPolicy(e env.Env) string {
	if e.IsLocal() {
		return "Never"
	}

	return "IfNotPresent"
}
