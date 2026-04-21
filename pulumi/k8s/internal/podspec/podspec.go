// Package podspec builds the Kubernetes PodSpec that is shared across the
// workload types (Deployment, Job, CronJob). All three need the same
// container body: image, resources derived from a profile, ambient stats
// env vars, configmap/secret environment, literal env values, and
// configmap-backed file mounts. Centralizing that here keeps each
// workload wrapper a thin shell around its Kubernetes parent resource.
package podspec

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/profile"
)

// Conf is the non-file configuration a workload consumes at runtime.
// ConfigMaps and Secrets are the names of objects in the target namespace
// whose entire set of keys is injected as environment variables via
// envFrom. Values is an inline map of additional literal env vars that
// are applied last (and therefore override any keys from the referenced
// ConfigMaps/Secrets that collide).
type Conf struct {
	ConfigMaps []string
	Secrets    []string
	Values     map[string]string
}

// Files mounts configmap data into the container filesystem.
//
// ConfigMaps maps an in-container mount path to a configmap reference.
// Each value is one of:
//
//   - "<configMapName>": mount every key of the ConfigMap into the path
//     as a directory of files.
//   - "<configMapName>/<key>": mount a single key of the ConfigMap at
//     the path using the Kubernetes subPath mechanism.
//
// Both forms share a single volume per ConfigMap, so the same
// ConfigMap can be mounted multiple times (as a directory and/or as
// individual keys) without defining multiple volumes.
type Files struct {
	ConfigMaps map[string]string
}

// Port is a container port declaration. Protocol is a Kubernetes port
// protocol (e.g. "TCP", "UDP", "SCTP").
type Port struct {
	Name          string
	ContainerPort int
	Protocol      string
}

// HTTPLivenessProbe configures an HTTP liveness probe for the container.
// Freq, when positive, is the probe period in seconds; zero selects the
// 15 second default.
type HTTPLivenessProbe struct {
	Path string
	Port int
	Freq int
}

// Args is the full set of inputs needed to construct a PodSpec.
//
// The pod's labels must be managed by the caller (Deployments/Jobs own
// both the selector and the template labels, and those must agree), so
// Args only handles the body inside the template. The ambient stats env
// vars are derived from Env, Namespace, and Name.
type Args struct {
	Env       env.Env
	Name      string
	Namespace string

	Image           string
	Tag             string
	ImagePullPolicy string
	Command         []string
	Args            []string

	Profile profile.Profile

	Conf  Conf
	Files Files
	Ports []Port

	HTTPLivenessProbe *HTTPLivenessProbe

	// EncourageColocation, when true, disables the preferred
	// podAntiAffinity that would otherwise steer pods with the same
	// `app=<Name>` label onto distinct nodes. The default (false)
	// applies the spread affinity to every workload built through
	// this package.
	EncourageColocation bool
}

// Build assembles a *corev1.PodSpecArgs from the resolved inputs. The
// caller is responsible for wrapping it in the appropriate
// PodTemplateSpec and attaching selectors/labels, since those vary by
// parent kind.
func Build(a Args) (*corev1.PodSpecArgs, error) {
	if a.Name == "" {
		return nil, fmt.Errorf("podspec: Name is required")
	}
	if a.Namespace == "" {
		return nil, fmt.Errorf("podspec: Namespace is required")
	}
	if a.Image == "" {
		return nil, fmt.Errorf("podspec: Image is required")
	}
	if a.Tag == "" {
		return nil, fmt.Errorf("podspec: Tag is required")
	}
	if a.ImagePullPolicy == "" {
		return nil, fmt.Errorf("podspec: ImagePullPolicy is required")
	}
	if err := a.Env.Validate(); err != nil {
		return nil, fmt.Errorf("podspec: %w", err)
	}

	container := &corev1.ContainerArgs{
		Name:            pulumi.String(a.Name),
		Image:           pulumi.String(a.Image + ":" + a.Tag),
		ImagePullPolicy: pulumi.String(a.ImagePullPolicy),
		Resources:       resources(a.Profile),
	}

	if len(a.Command) > 0 {
		container.Command = toStringArray(a.Command)
	}
	if len(a.Args) > 0 {
		container.Args = toStringArray(a.Args)
	}

	if len(a.Ports) > 0 {
		ports := make(corev1.ContainerPortArray, 0, len(a.Ports))
		for _, p := range a.Ports {
			ports = append(ports, &corev1.ContainerPortArgs{
				Name:          pulumi.String(p.Name),
				ContainerPort: pulumi.Int(p.ContainerPort),
				Protocol:      pulumi.String(p.Protocol),
			})
		}
		container.Ports = ports
	}

	envVars := append(statsEnv(a), literalEnv(a.Conf.Values)...)
	container.Env = envVars

	container.EnvFrom = envFrom(a.Conf)

	if a.HTTPLivenessProbe != nil {
		container.LivenessProbe = livenessProbe(a.HTTPLivenessProbe)
	}

	mounts, volumes := fileMounts(a.Files)
	container.VolumeMounts = mounts

	spec := &corev1.PodSpecArgs{
		Containers: corev1.ContainerArray{container},
	}
	if len(volumes) > 0 {
		spec.Volumes = volumes
	}
	if !a.EncourageColocation {
		spec.Affinity = spreadAcrossNodes(a.Name)
	}

	return spec, nil
}

func resources(p profile.Profile) *corev1.ResourceRequirementsArgs {
	return &corev1.ResourceRequirementsArgs{
		Requests: pulumi.StringMap{
			"cpu":    pulumi.Sprintf("%g", p.CPUCores.Min),
			"memory": pulumi.Sprintf("%dM", p.MemMB.Min),
		},
		Limits: pulumi.StringMap{
			"cpu":    pulumi.Sprintf("%g", p.CPUCores.Max),
			"memory": pulumi.Sprintf("%dM", p.MemMB.Max),
		},
	}
}

// statsEnv emits the ambient stats/telemetry env vars every workload
// inherits. The names are prefixed with Env.Prefix so the same process
// image can be scheduled into multiple environments without collision.
func statsEnv(a Args) corev1.EnvVarArray {
	prefix := a.Env.Prefix
	statsPrefix := fmt.Sprintf(
		"%s.%s.%s",
		strings.ToLower(prefix),
		a.Namespace,
		a.Name,
	)

	tags, _ := json.Marshal(map[string]string{
		"env":       a.Env.ID(),
		"namespace": a.Namespace,
		"service":   a.Name,
	})
	statsTags := string(tags)

	return corev1.EnvVarArray{
		&corev1.EnvVarArgs{
			Name:  pulumi.String(prefix + "_STATS_PREFIX"),
			Value: pulumi.String(statsPrefix),
		},
		&corev1.EnvVarArgs{
			Name:  pulumi.String(prefix + "_STATS_TAGS"),
			Value: pulumi.String(statsTags),
		},
		&corev1.EnvVarArgs{
			Name: pulumi.String(prefix + "_HOST"),
			ValueFrom: &corev1.EnvVarSourceArgs{
				FieldRef: &corev1.ObjectFieldSelectorArgs{
					FieldPath: pulumi.String("status.hostIP"),
				},
			},
		},
	}
}

func literalEnv(values map[string]string) corev1.EnvVarArray {
	if len(values) == 0 {
		return nil
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make(corev1.EnvVarArray, 0, len(values))
	for _, k := range keys {
		out = append(out, &corev1.EnvVarArgs{
			Name:  pulumi.String(k),
			Value: pulumi.String(values[k]),
		})
	}

	return out
}

func envFrom(c Conf) corev1.EnvFromSourceArray {
	if len(c.ConfigMaps) == 0 && len(c.Secrets) == 0 {
		return nil
	}

	out := make(corev1.EnvFromSourceArray, 0, len(c.ConfigMaps)+len(c.Secrets))
	for _, cm := range c.ConfigMaps {
		out = append(out, &corev1.EnvFromSourceArgs{
			ConfigMapRef: &corev1.ConfigMapEnvSourceArgs{
				Name: pulumi.String(cm),
			},
		})
	}
	for _, sec := range c.Secrets {
		out = append(out, &corev1.EnvFromSourceArgs{
			SecretRef: &corev1.SecretEnvSourceArgs{
				Name: pulumi.String(sec),
			},
		})
	}

	return out
}

func livenessProbe(p *HTTPLivenessProbe) *corev1.ProbeArgs {
	period := p.Freq
	if period <= 0 {
		period = 15
	}

	return &corev1.ProbeArgs{
		HttpGet: &corev1.HTTPGetActionArgs{
			Path: pulumi.String(p.Path),
			Port: pulumi.Int(p.Port),
		},
		InitialDelaySeconds: pulumi.Int(5),
		PeriodSeconds:       pulumi.Int(period),
	}
}

// fileMounts resolves Files into matching VolumeMounts and Volumes.
// A single Volume is emitted per referenced ConfigMap regardless of how
// many mounts reference it, so each configmap appears exactly once in
// the pod spec.
func fileMounts(files Files) (corev1.VolumeMountArray, corev1.VolumeArray) {
	if len(files.ConfigMaps) == 0 {
		return nil, nil
	}

	mountPaths := make([]string, 0, len(files.ConfigMaps))
	for k := range files.ConfigMaps {
		mountPaths = append(mountPaths, k)
	}
	sort.Strings(mountPaths)

	mounts := make(corev1.VolumeMountArray, 0, len(mountPaths))
	configMapNames := map[string]struct{}{}
	for _, mountPath := range mountPaths {
		spec := files.ConfigMaps[mountPath]
		name, subPath, hasSubPath := splitMount(spec)
		mount := &corev1.VolumeMountArgs{
			Name:      pulumi.String(name),
			MountPath: pulumi.String(mountPath),
		}
		if hasSubPath {
			mount.SubPath = pulumi.String(subPath)
		}
		mounts = append(mounts, mount)
		configMapNames[name] = struct{}{}
	}

	names := make([]string, 0, len(configMapNames))
	for n := range configMapNames {
		names = append(names, n)
	}
	sort.Strings(names)

	volumes := make(corev1.VolumeArray, 0, len(names))
	for _, n := range names {
		volumes = append(volumes, &corev1.VolumeArgs{
			Name: pulumi.String(n),
			ConfigMap: &corev1.ConfigMapVolumeSourceArgs{
				Name: pulumi.String(n),
			},
		})
	}

	return mounts, volumes
}

func splitMount(spec string) (name, subPath string, hasSubPath bool) {
	idx := strings.Index(spec, "/")
	if idx < 0 {
		return spec, "", false
	}

	return spec[:idx], spec[idx+1:], true
}

// spreadAcrossNodes produces a soft podAntiAffinity rule that encourages
// the scheduler to place replicas on distinct nodes. It is a preference,
// not a requirement, so single-node clusters (e.g. local) still schedule
// the pod successfully.
func spreadAcrossNodes(name string) *corev1.AffinityArgs {
	return &corev1.AffinityArgs{
		PodAntiAffinity: &corev1.PodAntiAffinityArgs{
			PreferredDuringSchedulingIgnoredDuringExecution: corev1.WeightedPodAffinityTermArray{
				&corev1.WeightedPodAffinityTermArgs{
					Weight: pulumi.Int(100),
					PodAffinityTerm: &corev1.PodAffinityTermArgs{
						TopologyKey: pulumi.String("kubernetes.io/hostname"),
						LabelSelector: &metav1.LabelSelectorArgs{
							MatchExpressions: metav1.LabelSelectorRequirementArray{
								&metav1.LabelSelectorRequirementArgs{
									Key:      pulumi.String("app"),
									Operator: pulumi.String("In"),
									Values:   pulumi.StringArray{pulumi.String(name)},
								},
							},
						},
					},
				},
			},
		},
	}
}

func toStringArray(in []string) pulumi.StringArray {
	out := make(pulumi.StringArray, 0, len(in))
	for _, v := range in {
		out = append(out, pulumi.String(v))
	}

	return out
}
