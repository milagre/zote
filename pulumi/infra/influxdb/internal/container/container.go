// Package container is the container-backed implementation of the influxdb
// backend interface defined in the parent influxdb package. It installs the
// upstream influxdb2 Helm chart and generates admin credentials.
package container

import (
	"fmt"
	"math"

	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/tokens"
	"github.com/milagre/zote/pulumi/profile"
)

const (
	chartName       = "influxdb2"
	chartRepository = "https://helm.influxdata.com"
	chartVersion    = "2.1.2"
)

var typeToken = tokens.Token("infra", "InfluxdbContainer")

// randomPasswordIgnoredArgs freezes the RandomPassword generation knobs after
// the resource exists so imported passwords (whose generator args may not
// match the live `result`) don't get rotated by a benign args diff.
var randomPasswordIgnoredArgs = []string{
	"length", "special", "upper", "lower", "numeric",
	"minLower", "minUpper", "minNumeric", "minSpecial", "overrideSpecial",
}

// Args is the caller-facing configuration for a container-backed influxdb.
// Fields the parent influxdb component fills in on the caller's behalf
// (instance identity, admin identity) are marked below; everything else
// is user-supplied.
type Args struct {
	// Env is the deploy environment (RotateSecrets drives optional RandomPassword keepers).
	Env env.Env
	// Namespace is the target Kubernetes namespace. Filled by the parent.
	Namespace string
	// Name is the influxdb instance name (used to derive the release name
	// "influxdb-<Name>"). Filled by the parent.
	Name string
	// Version is the influxdb image tag (without the "-alpine" suffix that
	// this component appends).
	Version string
	// Profile is the validated resource profile (CPU and memory).
	Profile profile.Profile
	// Organization is the influxdb admin organization (also used as the
	// default bucket name). Filled by the parent, which applies its own
	// fallback if the caller leaves it empty.
	Organization string
	// User is the influxdb admin username. Filled by the parent, which
	// applies its own fallback if the caller leaves it empty.
	User string
}

// Container installs influxdb via Helm and exposes the connection details
// required by the parent's ConfigMap and Secret wiring.
type Container struct {
	pulumi.ResourceState

	scheme pulumi.StringOutput
	host   pulumi.StringOutput
	port   pulumi.StringOutput
	org    pulumi.StringOutput
	bucket pulumi.StringOutput
	user   pulumi.StringOutput
	pass   pulumi.StringOutput
	token  pulumi.StringOutput
}

// New registers the container backend as a child component. parentName is
// the parent influxdb component's logical name; child resources are named
// uniformly off of it. influxParent is the outer Influxdb component; the
// Helm release is registered as its direct child alongside this component.
func New(ctx *pulumi.Context, parentName string, args *Args, influxParent pulumi.Resource) (*Container, error) {
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
	if args.Organization == "" {
		return nil, fmt.Errorf("%s: Organization is required (set by parent)", typeToken)
	}
	if args.User == "" {
		return nil, fmt.Errorf("%s: User is required (set by parent)", typeToken)
	}
	if err := args.Env.Validate(); err != nil {
		return nil, fmt.Errorf("%s: env: %w", typeToken, err)
	}

	comp := &Container{}
	if err := ctx.RegisterComponentResource(typeToken, parentName, comp, pulumi.Parent(influxParent)); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	releaseName := fmt.Sprintf("influxdb-%s", args.Name)

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
		return nil, fmt.Errorf("%s: generating admin password: %w", typeToken, err)
	}

	token, err := random.NewRandomPassword(ctx, parentName+"-token", &random.RandomPasswordArgs{
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
		return nil, fmt.Errorf("%s: generating admin token: %w", typeToken, err)
	}

	p := args.Profile
	cacheMax := int64(math.Floor(float64(p.MemMB.Max) * 0.6 * 1024 * 1024))

	values := pulumi.Map{
		"nameOverride":     pulumi.String(releaseName),
		"fullnameOverride": pulumi.String(releaseName),
		"adminUser": pulumi.Map{
			"organization": pulumi.String(args.Organization),
			"bucket":       pulumi.String(args.Organization),
			"user":         pulumi.String(args.User),
			"password":     password.Result,
			"token":        token.Result,
		},
		"image": pulumi.Map{
			"tag": pulumi.String(args.Version + "-alpine"),
		},
		"resources": pulumi.Map{
			"limits": pulumi.Map{
				"memory": pulumi.Sprintf("%dMi", p.MemMB.Max),
				"cpu":    pulumi.Float64(p.CPUCores.Max),
			},
			"requests": pulumi.Map{
				"memory": pulumi.Sprintf("%dMi", p.MemMB.Min),
				"cpu":    pulumi.Float64(p.CPUCores.Min),
			},
		},
		"extraEnvVars": pulumi.Array{
			pulumi.Map{
				"name":  pulumi.String("INFLUXD_STORAGE_CACHE_MAX_MEMORY_SIZE"),
				"value": pulumi.Int(cacheMax),
			},
		},
		"livenessProbe": pulumi.Map{
			"path":                pulumi.String("/health"),
			"scheme":              pulumi.String("HTTP"),
			"initialDelaySeconds": pulumi.Int(0),
			"periodSeconds":       pulumi.Int(10),
			"timeoutSeconds":      pulumi.Int(1),
			"failureThreshold":    pulumi.Int(3),
		},
		"readinessProbe": pulumi.Map{
			"path":                pulumi.String("/health"),
			"scheme":              pulumi.String("HTTP"),
			"initialDelaySeconds": pulumi.Int(0),
			"periodSeconds":       pulumi.Int(10),
			"timeoutSeconds":      pulumi.Int(1),
			"successThreshold":    pulumi.Int(1),
			"failureThreshold":    pulumi.Int(3),
		},
		"service": pulumi.Map{
			"type": pulumi.String("ClusterIP"),
		},
	}

	if _, err := helmv3.NewRelease(ctx, parentName, &helmv3.ReleaseArgs{
		Chart:     pulumi.String(chartName),
		Name:      pulumi.String(releaseName).ToStringPtrOutput(),
		Namespace: pulumi.String(args.Namespace).ToStringPtrOutput(),
		Version:   pulumi.String(chartVersion).ToStringPtrOutput(),
		Values:    values,
		RepositoryOpts: &helmv3.RepositoryOptsArgs{
			Repo: pulumi.String(chartRepository).ToStringPtrOutput(),
		},
	}, pulumi.Parent(influxParent)); err != nil {
		return nil, fmt.Errorf("%s: installing chart: %w", typeToken, err)
	}

	comp.scheme = pulumi.String("http").ToStringOutput()
	comp.host = pulumi.String(releaseName).ToStringOutput()
	comp.port = pulumi.String("80").ToStringOutput()
	comp.org = pulumi.String(args.Organization).ToStringOutput()
	comp.bucket = pulumi.String(args.Organization).ToStringOutput()
	comp.user = pulumi.String(args.User).ToStringOutput()
	comp.pass = password.Result
	comp.token = token.Result

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

// Scheme returns the URL scheme (always "http" for the container backend).
func (c *Container) Scheme() pulumi.StringOutput { return c.scheme }

// Host returns the in-cluster service hostname exposing influxdb.
func (c *Container) Host() pulumi.StringOutput { return c.host }

// Port returns the service port as a string.
func (c *Container) Port() pulumi.StringOutput { return c.port }

// Org returns the configured influxdb organization.
func (c *Container) Org() pulumi.StringOutput { return c.org }

// Bucket returns the default influxdb bucket to use. The container
// backend provisions a single bucket named after the organization so
// that a fresh install has one valid write target out of the box.
func (c *Container) Bucket() pulumi.StringOutput { return c.bucket }

// User returns the configured admin username.
func (c *Container) User() pulumi.StringOutput { return c.user }

// Pass returns the generated admin password.
func (c *Container) Pass() pulumi.StringOutput { return c.pass }

// Token returns the generated admin API token.
func (c *Container) Token() pulumi.StringOutput { return c.token }
