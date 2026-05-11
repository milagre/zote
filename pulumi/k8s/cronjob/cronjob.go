// Package cronjob is a CronJob built with the shared podspec.
package cronjob

import (
	"fmt"

	batchv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/batch/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/annotations"
	"github.com/milagre/zote/pulumi/k8s/internal/podspec"
	"github.com/milagre/zote/pulumi/profile"
	"github.com/milagre/zote/pulumi/tokens"
)

var typeToken = tokens.Token("k8s", "CronJob")

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

	Schedule string // k8s cron syntax; empty Timezone → Etc/UTC
	Timezone string
	Suspend  bool

	Conf  Conf
	Files Files
}

type CronJob struct {
	pulumi.ResourceState
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*CronJob, error) {
	if args == nil {
		return nil, fmt.Errorf("%s: args is required", typeToken)
	}
	if err := args.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	resourceName := tokens.Qualify(args.Namespace, name)

	comp := &CronJob{}
	if err := ctx.RegisterComponentResource(typeToken, resourceName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	spec, err := podspec.Build(podspec.Args{
		Env:                 args.Env,
		Name:                args.Name,
		Namespace:           args.Namespace,
		Image:               args.Image,
		Tag:                 args.Tag,
		ImagePullPolicy:     imagePullPolicy(args.Env),
		Command:             args.Command,
		Args:                args.Args,
		Profile:             args.Profile,
		Conf:                args.Conf,
		Files:               args.Files,
		EncourageColocation: true,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	spec.RestartPolicy = pulumi.String("Never")

	timezone := args.Timezone
	if timezone == "" {
		timezone = "Etc/UTC"
	}

	labels := pulumi.StringMap{"app": pulumi.String(args.Name)}

	if _, err := batchv1.NewCronJob(ctx, resourceName, &batchv1.CronJobArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(args.Name),
			Namespace:   pulumi.String(args.Namespace),
			Labels:      labels,
			Annotations: pulumi.StringMap{
				annotations.SkipAwaitKey: pulumi.String(annotations.SkipAwaitValueAll),
			},
		},
		Spec: &batchv1.CronJobSpecArgs{
			Schedule:                   pulumi.String(args.Schedule),
			TimeZone:                   pulumi.String(timezone),
			Suspend:                    pulumi.Bool(args.Suspend),
			ConcurrencyPolicy:          pulumi.String("Replace"),
			StartingDeadlineSeconds:    pulumi.Int(10),
			FailedJobsHistoryLimit:     pulumi.Int(5),
			SuccessfulJobsHistoryLimit: pulumi.Int(10),
			JobTemplate: &batchv1.JobTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:   pulumi.String(args.Name),
					Labels: labels,
				},
				Spec: &batchv1.JobSpecArgs{
					BackoffLimit:            pulumi.Int(2),
					TtlSecondsAfterFinished: pulumi.Int(3600),
					Template: &corev1.PodTemplateSpecArgs{
						Metadata: &metav1.ObjectMetaArgs{
							Name:   pulumi.String(args.Name),
							Labels: labels,
						},
						Spec: spec,
					},
				},
			},
		},
	}, pulumi.Parent(comp)); err != nil {
		return nil, fmt.Errorf("%s: cronjob: %w", typeToken, err)
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
	if a.Schedule == "" {
		return fmt.Errorf("Schedule is required")
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
