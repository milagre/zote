// Package backup schedules dumps of a data store into object storage.
//
// Each run is one Pod: an init container produces a single compressed file
// at /dump/dump using the store's own client image, then the uploader writes
// it under <name>/<timestamp><ext> with a private ACL, and fails the run
// unless an unauthenticated GET of the object and of the bucket listing both
// return 403. A store that is publicly readable therefore surfaces as a
// failed Job rather than going unnoticed, and the exposed object is deleted.
package backup

import (
	"fmt"

	batchv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/batch/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/svc/objectstorage"
	"github.com/milagre/zote/pulumi/util/annotations"
	"github.com/milagre/zote/pulumi/util/labels"
	"github.com/milagre/zote/pulumi/util/profile"
	"github.com/milagre/zote/pulumi/util/stringdata"
)

// uploaderImage predates the AWS CLI's default request checksums, which
// S3-compatible stores do not all accept; the checksum env vars below cover
// a bump past it. The privacy checks rely on the image's curl.
const uploaderImage = "amazon/aws-cli:2.22.35"

const dumpDir = "/dump"

const uploadScript = `set -euo pipefail
key="$BACKUP_PREFIX/$(date -u +%Y%m%dT%H%M%SZ)$BACKUP_EXT"
aws s3 cp /dump/dump "s3://$BACKUP_BUCKET/$key" --endpoint-url "$BACKUP_S3_URL" --acl private --only-show-errors

anonymous() {
  curl -s -o /dev/null -w '%{http_code}' "$BACKUP_S3_URL/$BACKUP_BUCKET/$1"
}

code=$(anonymous "$key")
if [ "$code" = 200 ]; then
  aws s3 rm "s3://$BACKUP_BUCKET/$key" --endpoint-url "$BACKUP_S3_URL" --only-show-errors
  echo "$key was publicly readable and has been deleted" >&2
  exit 1
fi
if [ "$code" != 403 ]; then
  echo "could not confirm $key is private: anonymous GET returned HTTP $code" >&2
  exit 1
fi

code=$(anonymous "")
if [ "$code" != 403 ]; then
  echo "could not confirm bucket $BACKUP_BUCKET is private: anonymous listing returned HTTP $code" >&2
  exit 1
fi

echo "uploaded $key"
`

// Spec is what a store's owner supplies to have it backed up. Bucket is a
// bucket name as configured on Storage.
type Spec struct {
	Storage  objectstorage.ObjectStorage
	Bucket   string
	Schedule string // k8s cron syntax, Etc/UTC
	Profile  profile.Profile
}

// Dumper is the store-specific half of a run. Script runs under Shell in
// Image with Env set, and must leave exactly one file at /dump/dump,
// compressed by the dumper since the uploader image has no compressor. Ext is
// that file's extension.
type Dumper struct {
	Image  string
	Shell  string
	Script string
	Env    map[string]pulumi.StringOutput
	Ext    string
}

// Args names the CronJob (<Name>-backup) and its Secrets in Namespace. Name
// also prefixes every object key.
type Args struct {
	Namespace string
	Name      string
	Spec      Spec
	Dumper    Dumper
}

func (s Spec) Validate() error {
	if s.Schedule == "" {
		return fmt.Errorf("schedule is required")
	}
	if s.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	if _, err := s.Storage.ProvisionedBucket(s.Bucket); err != nil {
		return fmt.Errorf("bucket: %w", err)
	}
	if s.Profile.MemMB.Min <= 0 || s.Profile.CPUCores.Min <= 0 {
		return fmt.Errorf("profile is required")
	}

	return nil
}

// New registers the backup CronJob and the Secrets it reads, parented to parent.
// The dump and upload containers each see only their own Secret.
func New(ctx *pulumi.Context, resourceName string, parent pulumi.Resource, a Args) error {
	if a.Namespace == "" {
		return fmt.Errorf("backup: Namespace is required")
	}
	if a.Name == "" {
		return fmt.Errorf("backup: Name is required")
	}
	if err := a.Spec.Validate(); err != nil {
		return fmt.Errorf("backup: %w", err)
	}

	bucket, err := a.Spec.Storage.ProvisionedBucket(a.Spec.Bucket)
	if err != nil {
		return fmt.Errorf("backup: %w", err)
	}

	name := a.Name + "-backup"

	dumpEnv, err := secretEnvFrom(ctx, resourceName+"-backup-dump", parent, a.Namespace, name+"-dump", a.Dumper.Env)
	if err != nil {
		return fmt.Errorf("backup: dump secret: %w", err)
	}

	uploadEnv, err := secretEnvFrom(ctx, resourceName+"-backup-upload", parent, a.Namespace, name+"-upload", map[string]pulumi.StringOutput{
		"AWS_ACCESS_KEY_ID":     a.Spec.Storage.Creds.AccessKey,
		"AWS_SECRET_ACCESS_KEY": a.Spec.Storage.Creds.SecretKey,
	})
	if err != nil {
		return fmt.Errorf("backup: upload secret: %w", err)
	}

	mounts := corev1.VolumeMountArray{
		&corev1.VolumeMountArgs{Name: pulumi.String("dump"), MountPath: pulumi.String(dumpDir)},
	}
	resources := &corev1.ResourceRequirementsArgs{
		Requests: pulumi.StringMap{
			"cpu":    pulumi.Sprintf("%g", a.Spec.Profile.CPUCores.Min),
			"memory": pulumi.Sprintf("%dM", a.Spec.Profile.MemMB.Min),
		},
		Limits: pulumi.StringMap{
			"cpu":    pulumi.Sprintf("%g", a.Spec.Profile.CPUCores.Max),
			"memory": pulumi.Sprintf("%dM", a.Spec.Profile.MemMB.Max),
		},
	}

	podLabels := labels.Pod(a.Namespace, name)

	if _, err := batchv1.NewCronJob(ctx, resourceName+"-backup", &batchv1.CronJobArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(name),
			Namespace: pulumi.String(a.Namespace),
			Labels:    podLabels,
			Annotations: pulumi.StringMap{
				annotations.WaitForKey: pulumi.String(annotations.WaitForValueImmediate),
			},
		},
		Spec: &batchv1.CronJobSpecArgs{
			Schedule: pulumi.String(a.Spec.Schedule),
			TimeZone: pulumi.String("Etc/UTC"),
			// A run still going when the next is due is a slow dump, not a
			// stuck one; replacing it would lose both.
			ConcurrencyPolicy:          pulumi.String("Forbid"),
			StartingDeadlineSeconds:    pulumi.Int(3600),
			FailedJobsHistoryLimit:     pulumi.Int(5),
			SuccessfulJobsHistoryLimit: pulumi.Int(3),
			JobTemplate: &batchv1.JobTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{Labels: podLabels},
				Spec: &batchv1.JobSpecArgs{
					BackoffLimit: pulumi.Int(2),
					Template: &corev1.PodTemplateSpecArgs{
						Metadata: &metav1.ObjectMetaArgs{Labels: podLabels},
						Spec: &corev1.PodSpecArgs{
							RestartPolicy: pulumi.String("Never"),
							Volumes: corev1.VolumeArray{
								&corev1.VolumeArgs{Name: pulumi.String("dump"), EmptyDir: &corev1.EmptyDirVolumeSourceArgs{}},
							},
							InitContainers: corev1.ContainerArray{
								&corev1.ContainerArgs{
									Name:         pulumi.String("dump"),
									Image:        pulumi.String(a.Dumper.Image),
									Command:      pulumi.StringArray{pulumi.String(a.Dumper.Shell), pulumi.String("-c"), pulumi.String(a.Dumper.Script)},
									EnvFrom:      dumpEnv,
									VolumeMounts: mounts,
									Resources:    resources,
								},
							},
							Containers: corev1.ContainerArray{
								&corev1.ContainerArgs{
									Name:    pulumi.String("upload"),
									Image:   pulumi.String(uploaderImage),
									Command: pulumi.StringArray{pulumi.String("/bin/bash"), pulumi.String("-c"), pulumi.String(uploadScript)},
									Env: corev1.EnvVarArray{
										envVar("BACKUP_S3_URL", a.Spec.Storage.S3.URL),
										envVar("BACKUP_BUCKET", pulumi.String(bucket)),
										envVar("BACKUP_PREFIX", pulumi.String(a.Name)),
										envVar("BACKUP_EXT", pulumi.String(a.Dumper.Ext)),
										envVar("AWS_DEFAULT_REGION", pulumi.String("us-east-1")),
										envVar("AWS_REQUEST_CHECKSUM_CALCULATION", pulumi.String("when_required")),
										envVar("AWS_RESPONSE_CHECKSUM_VALIDATION", pulumi.String("when_required")),
									},
									EnvFrom:      uploadEnv,
									VolumeMounts: mounts,
									Resources:    resources,
								},
							},
						},
					},
				},
			},
		},
	}, pulumi.Parent(parent), pulumi.DependsOn(a.Spec.Storage.Deps)); err != nil {
		return fmt.Errorf("backup: cronjob: %w", err)
	}

	return nil
}

func envVar(name string, value pulumi.StringInput) *corev1.EnvVarArgs {
	return &corev1.EnvVarArgs{Name: pulumi.String(name), Value: value}
}

func secretEnvFrom(
	ctx *pulumi.Context,
	resourceName string,
	parent pulumi.Resource,
	namespace string,
	name string,
	data map[string]pulumi.StringOutput,
) (corev1.EnvFromSourceArray, error) {
	sec, err := corev1.NewSecret(ctx, resourceName, &corev1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(name),
			Namespace: pulumi.String(namespace),
		},
		Type: pulumi.String("Opaque"),
		Data: stringdata.SecretData(data),
	}, pulumi.Parent(parent))
	if err != nil {
		return nil, fmt.Errorf("creating secret: %w", err)
	}

	return corev1.EnvFromSourceArray{
		&corev1.EnvFromSourceArgs{SecretRef: &corev1.SecretEnvSourceArgs{Name: sec.Metadata.Name()}},
	}, nil
}
