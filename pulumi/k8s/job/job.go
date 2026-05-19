// Package job is a Job built with the shared podspec.
package job

import (
	"fmt"

	batchv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/batch/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/k8s/internal/podspec"
	"github.com/milagre/zote/pulumi/util/annotations"
	"github.com/milagre/zote/pulumi/util/labels"
	"github.com/milagre/zote/pulumi/util/profile"
	"github.com/milagre/zote/pulumi/util/tokens"
)

var typeToken = tokens.Token("k8s", "Job")

const defaultBackoffLimit = 100000

type (
	Conf  = podspec.Conf
	Files = podspec.Files
)

type Args struct {
	Env       env.Env
	Namespace string
	Name      string

	Image   string
	Tag     string
	Command []string
	Args    []string
	Profile profile.Profile

	Attempts *int // nil → defaultBackoffLimit
	Wait     bool

	Conf  Conf
	Files Files
}

type Job struct {
	pulumi.ResourceState
}

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

	podLabels := labels.Pod(args.Namespace, args.Name)

	// When Wait is false we want to skip the builtin "wait for Job to Succeed"
	// awaiter while *still* awaiting delete on replace — Job spec is largely
	// immutable so any template change replaces, and delete-await keeps the
	// new POST from racing the old object's tombstone (AlreadyExists).
	// pulumi.com/skipAwait="true" can't be used here: for batch/v1.Job the
	// provider treats it as skip-create-AND-skip-delete (allowsSkipAwaitWithDelete).
	// pulumi.com/waitFor with a trivially-satisfied JSONPath bypasses only the
	// per-kind awaiter (custom=true in ReadyCondition).
	var metaAnnotations pulumi.StringMap
	if !args.Wait {
		metaAnnotations = pulumi.StringMap{
			annotations.WaitForKey: pulumi.String(annotations.WaitForValueImmediate),
		}
	}

	if _, err := batchv1.NewJob(ctx, resourceName, &batchv1.JobArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(args.Name),
			Namespace:   pulumi.String(args.Namespace),
			Labels:      podLabels,
			Annotations: metaAnnotations,
		},
		Spec: &batchv1.JobSpecArgs{
			BackoffLimit: pulumi.Int(backoff),
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:   pulumi.String(args.Name),
					Labels: podLabels,
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
