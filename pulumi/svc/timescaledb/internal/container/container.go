// Package container is in-cluster TimescaleDB (StatefulSet, Service, libpq-style secret).
package container

import (
	"fmt"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/util/annotations"
	"github.com/milagre/zote/pulumi/util/labels"
	"github.com/milagre/zote/pulumi/util/profile"
	"github.com/milagre/zote/pulumi/util/tokens"
)

const (
	image   = "timescale/timescaledb:latest-pg17"
	pgPort  = 5432
	storage = "10Gi"
)

var typeToken = tokens.Token("svc", "TimescaledbContainer")

var randomPasswordIgnoredArgs = []string{
	"length", "special", "upper", "lower", "numeric",
	"minLower", "minUpper", "minNumeric", "minSpecial", "overrideSpecial",
}

type Args struct {
	Env       env.Env
	Namespace string
	Name      string
	Profile   profile.Profile
	User      string
	Database  string
}

type Container struct {
	pulumi.ResourceState

	StatefulSet *appsv1.StatefulSet
	Service     *corev1.Service
	PVC         *corev1.PersistentVolumeClaim
	PGSecret    *corev1.Secret

	scheme   pulumi.StringOutput
	host     pulumi.StringOutput
	port     pulumi.StringOutput
	user     pulumi.StringOutput
	pass     pulumi.StringOutput
	database pulumi.StringOutput
}

func New(ctx *pulumi.Context, parentName string, args *Args, opts ...pulumi.ResourceOption) (*Container, error) {
	if args == nil {
		return nil, fmt.Errorf("%s: args is required", typeToken)
	}
	if args.Name == "" {
		return nil, fmt.Errorf("%s: Name is required", typeToken)
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("%s: Namespace is required", typeToken)
	}
	if args.User == "" {
		return nil, fmt.Errorf("%s: User is required (resolved by parent)", typeToken)
	}
	if args.Database == "" {
		return nil, fmt.Errorf("%s: Database is required (resolved by parent)", typeToken)
	}
	if err := args.Env.Validate(); err != nil {
		return nil, fmt.Errorf("%s: env: %w", typeToken, err)
	}

	comp := &Container{}
	if err := ctx.RegisterComponentResource(typeToken, parentName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	releaseName := fmt.Sprintf("timescaledb-%s", args.Name)
	podLabels := labels.Pod("timescaledb", args.Name)
	ns := pulumi.String(args.Namespace)

	password, err := random.NewRandomPassword(ctx, parentName+"-password", &random.RandomPasswordArgs{
		Length:     pulumi.Int(32),
		Numeric:    pulumi.Bool(true),
		Upper:      pulumi.Bool(true),
		Lower:      pulumi.Bool(true),
		Special:    pulumi.Bool(false),
		MinNumeric: pulumi.Int(8),
		MinLower:   pulumi.Int(8),
		MinUpper:   pulumi.Int(8),
		Keepers:    args.Env.RandomKeepers(nil),
	},
		pulumi.Parent(comp),
		pulumi.IgnoreChanges(randomPasswordIgnoredArgs),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: generating password: %w", typeToken, err)
	}

	pvc, err := corev1.NewPersistentVolumeClaim(ctx, parentName+"-pvc", &corev1.PersistentVolumeClaimArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(releaseName + "-pvc"),
			Namespace: ns,
		},
		Spec: &corev1.PersistentVolumeClaimSpecArgs{
			AccessModes: pulumi.StringArray{pulumi.String("ReadWriteOnce")},
			Resources: &corev1.VolumeResourceRequirementsArgs{
				Requests: pulumi.StringMap{"storage": pulumi.String(storage)},
			},
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: pvc: %w", typeToken, err)
	}

	sts, err := appsv1.NewStatefulSet(ctx, parentName, &appsv1.StatefulSetArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(releaseName),
			Namespace: ns,
			Labels:    podLabels,
			Annotations: pulumi.StringMap{
				annotations.SkipAwaitKey: pulumi.String(annotations.SkipAwaitValueAll),
			},
		},
		Spec: &appsv1.StatefulSetSpecArgs{
			ServiceName: pulumi.String(releaseName),
			Replicas:    pulumi.Int(1),
			Selector: &metav1.LabelSelectorArgs{
				MatchLabels: podLabels,
			},
			Template: &corev1.PodTemplateSpecArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Labels: podLabels,
				},
				Spec: &corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						&corev1.ContainerArgs{
							Name:  pulumi.String("timescaledb"),
							Image: pulumi.String(image),
							Env: corev1.EnvVarArray{
								&corev1.EnvVarArgs{Name: pulumi.String("POSTGRES_USER"), Value: pulumi.String(args.User)},
								&corev1.EnvVarArgs{Name: pulumi.String("POSTGRES_PASSWORD"), Value: password.Result},
								&corev1.EnvVarArgs{Name: pulumi.String("POSTGRES_DB"), Value: pulumi.String(args.Database)},
								&corev1.EnvVarArgs{Name: pulumi.String("PGDATA"), Value: pulumi.String("/var/lib/postgresql/data/pgdata")},
							},
							Ports: corev1.ContainerPortArray{
								&corev1.ContainerPortArgs{ContainerPort: pulumi.Int(pgPort)},
							},
							VolumeMounts: corev1.VolumeMountArray{
								&corev1.VolumeMountArgs{
									Name:      pulumi.String("timescale-storage"),
									MountPath: pulumi.String("/var/lib/postgresql/data"),
								},
							},
							Resources: &corev1.ResourceRequirementsArgs{
								Limits: pulumi.StringMap{
									"cpu":    pulumi.Sprintf("%g", args.Profile.CPUCores.Max),
									"memory": pulumi.Sprintf("%dMi", args.Profile.MemMB.Max),
								},
								Requests: pulumi.StringMap{
									"cpu":    pulumi.Sprintf("%g", args.Profile.CPUCores.Min),
									"memory": pulumi.Sprintf("%dMi", args.Profile.MemMB.Min),
								},
							},
						},
					},
					Volumes: corev1.VolumeArray{
						&corev1.VolumeArgs{
							Name: pulumi.String("timescale-storage"),
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSourceArgs{
								ClaimName: pvc.Metadata.Name().Elem(),
							},
						},
					},
				},
			},
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: statefulset: %w", typeToken, err)
	}

	svc, err := corev1.NewService(ctx, parentName, &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(releaseName),
			Namespace: ns,
			Labels:    podLabels,
		},
		Spec: &corev1.ServiceSpecArgs{
			Type:     pulumi.String("ClusterIP"),
			Selector: podLabels,
			Ports: corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Protocol:   pulumi.String("TCP"),
					Port:       pulumi.Int(pgPort),
					TargetPort: pulumi.Int(pgPort),
				},
			},
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: service: %w", typeToken, err)
	}

	// A libpq-style secret alongside the instance so workloads that rely on
	// standard env-var names can mount it directly. Separate from the
	// cfg-prefixed client Secret the parent component emits.
	pgSecret, err := corev1.NewSecret(ctx, parentName+"-pg", &corev1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(releaseName + "-secret"),
			Namespace:   ns,
			Annotations: pulumi.StringMap{"pulumi.com/patchForce": pulumi.String("true")},
		},
		Type: pulumi.String("Opaque"),
		StringData: pulumi.All(
			pulumi.String(releaseName),
			pulumi.Sprintf("%d", pgPort),
			pulumi.String(args.Database),
			pulumi.String(args.User),
			password.Result,
		).ApplyT(func(xs []any) map[string]string {
			return map[string]string{
				"PGHOST":     xs[0].(string),
				"PGPORT":     xs[1].(string),
				"PGDATABASE": xs[2].(string),
				"PGUSER":     xs[3].(string),
				"PGPASSWORD": xs[4].(string),
			}
		}).(pulumi.StringMapOutput),
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: pg secret: %w", typeToken, err)
	}

	comp.StatefulSet = sts
	comp.Service = svc
	comp.PVC = pvc
	comp.PGSecret = pgSecret

	comp.scheme = pulumi.String("postgresql").ToStringOutput()
	comp.host = pulumi.String(releaseName).ToStringOutput()
	comp.port = pulumi.Sprintf("%d", pgPort).ToStringOutput()
	comp.user = pulumi.String(args.User).ToStringOutput()
	comp.pass = password.Result
	comp.database = pulumi.String(args.Database).ToStringOutput()

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

func (c *Container) Scheme() pulumi.StringOutput   { return c.scheme }
func (c *Container) Host() pulumi.StringOutput     { return c.host }
func (c *Container) Port() pulumi.StringOutput     { return c.port }
func (c *Container) User() pulumi.StringOutput     { return c.user }
func (c *Container) Pass() pulumi.StringOutput     { return c.pass }
func (c *Container) Database() pulumi.StringOutput { return c.database }
