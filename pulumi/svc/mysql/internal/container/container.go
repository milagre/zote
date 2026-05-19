// Package container: mysql YAML container Spec and Tier profiles (shared by Spec and Args), plus in-cluster StatefulSet MySQL (ordinal primary/replica init).
package container

import (
	"fmt"
	"math"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/util/annotations"
	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/util/labels"
	"github.com/milagre/zote/pulumi/util/profile"
	"github.com/milagre/zote/pulumi/util/stringdata"
	"github.com/milagre/zote/pulumi/util/tokens"
)

const (
	mysqlPort = 3306
	storage   = "10Gi"

	imageMysql = "mysql"
	imageInit  = "bash:5"
)

var typeToken = tokens.Token("svc", "MysqlContainer")

var randomPasswordIgnoredArgs = []string{
	"length", "special", "upper", "lower", "numeric",
	"minLower", "minUpper", "minNumeric", "minSpecial", "overrideSpecial",
}

type Args struct {
	Env       env.Env
	Namespace string
	Name      string
	Version   string
	Container *Spec
	Database  string
	Username  string
}

type Container struct {
	pulumi.ResourceState

	username pulumi.StringOutput
	hostname pulumi.StringOutput
	port     pulumi.StringOutput
	password pulumi.StringOutput
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
	if args.Version == "" {
		return nil, fmt.Errorf("%s: Version is required", typeToken)
	}
	if args.Database == "" {
		return nil, fmt.Errorf("%s: Database is required", typeToken)
	}
	if args.Username == "" {
		return nil, fmt.Errorf("%s: Username is required", typeToken)
	}
	if err := args.Env.Validate(); err != nil {
		return nil, fmt.Errorf("%s: env: %w", typeToken, err)
	}

	if args.Container == nil {
		return nil, fmt.Errorf("%s: container config is required", typeToken)
	}
	if err := args.Container.Validate(); err != nil {
		return nil, fmt.Errorf("%s: container config: %w", typeToken, err)
	}

	primary, err := profile.New(args.Container.Primary.Profile)
	if err != nil {
		return nil, fmt.Errorf("%s: container.primary.profile: %w", typeToken, err)
	}

	comp := &Container{}
	if err := ctx.RegisterComponentResource(typeToken, parentName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	releaseName := args.Name
	podLabels := labels.Pod("mysql", args.Name)

	password, err := random.NewRandomPassword(ctx, parentName+"-password", &random.RandomPasswordArgs{
		Length:          pulumi.Int(64),
		Numeric:         pulumi.Bool(true),
		Upper:           pulumi.Bool(true),
		Lower:           pulumi.Bool(true),
		Special:         pulumi.Bool(false),
		MinNumeric:      pulumi.Int(8),
		MinLower:        pulumi.Int(8),
		MinUpper:        pulumi.Int(8),
		OverrideSpecial: pulumi.String("$%&*()-_=+[]{}<>:?"),
		Keepers:         args.Env.RandomKeepers(nil),
	},
		pulumi.Parent(comp),
		pulumi.IgnoreChanges(randomPasswordIgnoredArgs),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: generating password: %w", typeToken, err)
	}

	cfgName := "cfg-" + releaseName
	patchForce := pulumi.StringMap{"pulumi.com/patchForce": pulumi.String("true")}
	cfgCM, err := corev1.NewConfigMap(ctx, parentName+"-config", &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(cfgName),
			Namespace:   pulumi.String(args.Namespace),
			Labels:      podLabels,
			Annotations: patchForce,
		},
		Data: pulumi.StringMap{
			"primary.cnf": pulumi.String(primaryCnf(primary)),
			"replica.cnf": pulumi.String(replicaCnf(primary)),
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: config configmap: %w", typeToken, err)
	}

	passwordSecret, err := corev1.NewSecret(ctx, parentName+"-password", &corev1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(cfgName),
			Namespace:   pulumi.String(args.Namespace),
			Labels:      podLabels,
			Annotations: patchForce,
		},
		Type: pulumi.String("Opaque"),
		Data: stringdata.SecretData(map[string]pulumi.StringOutput{
			"MYSQL_PASSWORD": password.Result,
		}),
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: password secret: %w", typeToken, err)
	}

	if _, err := corev1.NewService(ctx, parentName, &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(releaseName),
			Namespace: pulumi.String(args.Namespace),
			Labels:    podLabels,
		},
		Spec: &corev1.ServiceSpecArgs{
			Selector: podLabels,
			Ports: corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Name: pulumi.String("mysql"),
					Port: pulumi.Int(mysqlPort),
				},
			},
		},
	}, pulumi.Parent(comp)); err != nil {
		return nil, fmt.Errorf("%s: service: %w", typeToken, err)
	}

	if _, err := appsv1.NewStatefulSet(ctx, parentName, &appsv1.StatefulSetArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(releaseName),
			Namespace:   pulumi.String(args.Namespace),
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
				Spec: podSpec(primary, args, cfgCM, passwordSecret),
			},
			VolumeClaimTemplates: corev1.PersistentVolumeClaimTypeArray{
				&corev1.PersistentVolumeClaimTypeArgs{
					Metadata: &metav1.ObjectMetaArgs{
						Name: pulumi.String("data"),
					},
					Spec: &corev1.PersistentVolumeClaimSpecArgs{
						AccessModes: pulumi.StringArray{pulumi.String("ReadWriteOnce")},
						Resources: &corev1.VolumeResourceRequirementsArgs{
							Requests: pulumi.StringMap{"storage": pulumi.String(storage)},
						},
					},
				},
			},
		},
	}, pulumi.Parent(comp)); err != nil {
		return nil, fmt.Errorf("%s: statefulset: %w", typeToken, err)
	}

	comp.username = pulumi.String(args.Username).ToStringOutput()
	comp.hostname = pulumi.String(releaseName + "." + args.Namespace + ".svc.cluster.local").ToStringOutput()
	comp.port = pulumi.Sprintf("%d", mysqlPort).ToStringOutput()
	comp.password = password.Result

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

func (c *Container) Username() pulumi.StringOutput { return c.username }
func (c *Container) Hostname() pulumi.StringOutput { return c.hostname }
func (c *Container) Port() pulumi.StringOutput     { return c.port }
func (c *Container) Password() pulumi.StringOutput { return c.password }

func podSpec(primary profile.Profile, args *Args, cfgCM *corev1.ConfigMap, passwordSecret *corev1.Secret) *corev1.PodSpecArgs {
	return &corev1.PodSpecArgs{
		Volumes: corev1.VolumeArray{
			&corev1.VolumeArgs{
				Name:     pulumi.String("conf"),
				EmptyDir: &corev1.EmptyDirVolumeSourceArgs{},
			},
			&corev1.VolumeArgs{
				Name: pulumi.String("config-map"),
				ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
					Name: cfgCM.Metadata.Name(),
				},
			},
		},
		InitContainers: corev1.ContainerArray{
			&corev1.ContainerArgs{
				Name:    pulumi.String("init-mysql"),
				Image:   pulumi.String(imageInit),
				Command: pulumi.StringArray{pulumi.String("bash"), pulumi.String("-c"), pulumi.String(initScript)},
				VolumeMounts: corev1.VolumeMountArray{
					&corev1.VolumeMountArgs{
						Name:      pulumi.String("conf"),
						MountPath: pulumi.String("/mnt/conf.d"),
					},
					&corev1.VolumeMountArgs{
						Name:      pulumi.String("config-map"),
						MountPath: pulumi.String("/mnt/config-map"),
					},
				},
			},
		},
		Containers: corev1.ContainerArray{
			&corev1.ContainerArgs{
				Name:  pulumi.String("mysql"),
				Image: pulumi.String(imageMysql + ":" + args.Version),
				Ports: corev1.ContainerPortArray{
					&corev1.ContainerPortArgs{
						Name:          pulumi.String("mysql"),
						ContainerPort: pulumi.Int(mysqlPort),
					},
				},
				Env: corev1.EnvVarArray{
					&corev1.EnvVarArgs{Name: pulumi.String("MYSQL_ALLOW_EMPTY_PASSWORD"), Value: pulumi.String("yes")},
					&corev1.EnvVarArgs{Name: pulumi.String("MYSQL_ROOT_HOST"), Value: pulumi.String("127.0.0.1")},
					&corev1.EnvVarArgs{Name: pulumi.String("MYSQL_DATABASE"), Value: pulumi.String(args.Database)},
					&corev1.EnvVarArgs{Name: pulumi.String("MYSQL_USER"), Value: pulumi.String(args.Username)},
				},
				EnvFrom: corev1.EnvFromSourceArray{
					&corev1.EnvFromSourceArgs{
						SecretRef: &corev1.SecretEnvSourceArgs{
							Name: passwordSecret.Metadata.Name(),
						},
					},
				},
				Resources: &corev1.ResourceRequirementsArgs{
					Requests: pulumi.StringMap{
						"cpu":    pulumi.Sprintf("%g", primary.CPUCores.Min),
						"memory": pulumi.Sprintf("%dM", primary.MemMB.Min),
					},
					Limits: pulumi.StringMap{
						"cpu":    pulumi.Sprintf("%g", primary.CPUCores.Max),
						"memory": pulumi.Sprintf("%dM", primary.MemMB.Max),
					},
				},
				VolumeMounts: corev1.VolumeMountArray{
					&corev1.VolumeMountArgs{
						Name:      pulumi.String("data"),
						MountPath: pulumi.String("/var/lib/mysql"),
						SubPath:   pulumi.String("mysql"),
					},
					&corev1.VolumeMountArgs{
						Name:      pulumi.String("conf"),
						MountPath: pulumi.String("/etc/mysql/conf.d"),
					},
				},
				LivenessProbe: &corev1.ProbeArgs{
					Exec: &corev1.ExecActionArgs{
						Command: pulumi.StringArray{pulumi.String("mysqladmin"), pulumi.String("ping")},
					},
					InitialDelaySeconds: pulumi.Int(30),
					TimeoutSeconds:      pulumi.Int(5),
					PeriodSeconds:       pulumi.Int(10),
				},
				ReadinessProbe: &corev1.ProbeArgs{
					Exec: &corev1.ExecActionArgs{
						Command: pulumi.StringArray{pulumi.String("mysqladmin"), pulumi.String("ping")},
					},
					InitialDelaySeconds: pulumi.Int(5),
					TimeoutSeconds:      pulumi.Int(1),
					PeriodSeconds:       pulumi.Int(2),
				},
			},
		},
	}
}

func primaryCnf(p profile.Profile) string {
	return fmt.Sprintf("[mysqld]\nlog-bin\nmax_connections=100\ninnodb_buffer_pool_size=%dM\n",
		int(math.Floor(float64(p.MemMB.Max)*0.5)))
}

func replicaCnf(p profile.Profile) string {
	return fmt.Sprintf("[mysqld]\nsuper-read-only\nmax_connections=100\ninnodb_buffer_pool_size=%dM\n",
		int(math.Floor(float64(p.MemMB.Max)*0.5)))
}

const initScript = `set -ex

[[ ` + "`" + `hostname` + "`" + ` =~ -([0-9]+)$ ]] || exit 1
ordinal=${BASH_REMATCH[1]}
echo [mysqld] > /mnt/conf.d/server-id.cnf
echo server-id=$((100 + $ordinal)) >> /mnt/conf.d/server-id.cnf

if [[ $ordinal -eq 0 ]]; then
    cp /mnt/config-map/primary.cnf /mnt/conf.d/
else
    cp /mnt/config-map/replica.cnf /mnt/conf.d/
fi
`
