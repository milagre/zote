// Package container is the influxdb container backend: influxdb2 Helm chart
// with admin password and token from RandomPasswords wired into chart values.
//
// Chart *-auth may not match those values after Helm's lookup()-based reuse
// of an existing Secret, so the parent client Secret (keys under Env.Prefix) can diverge from *-auth.
//
// Admin RandomPasswords skip Env.RotateSecrets in keepers and use IgnoreChanges("*"):
// replacing them from stack config alone would not realign Influx's persisted admin state.
package container

import (
	"fmt"
	"math"

	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/profile"
	"github.com/milagre/zote/pulumi/tokens"
)

const (
	chartName       = "influxdb2"
	chartRepository = "https://helm.influxdata.com"
	chartVersion    = "2.1.2"
)

var typeToken = tokens.Token("infra", "InfluxdbContainer")

// Freezes admin RandomPassword inputs after create; see package doc.
var adminRandomPasswordIgnoreChanges = []string{"*"}

// Args configures the container backend; parent fills Namespace, Name, Org, User.
type Args struct {
	Env          env.Env
	Namespace    string
	Name         string
	Version      string
	Profile      profile.Profile
	Organization string
	User         string
}

// Container is the Helm influxdb2 backend; outputs feed the parent ConfigMap/Secret.
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

// New registers this component; the Helm release is also a child of influxParent.
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
		Keepers:         args.Env.RandomKeepers(nil, env.SupportsRotation(false)),
	},
		pulumi.Parent(comp),
		pulumi.IgnoreChanges(adminRandomPasswordIgnoreChanges),
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
		Keepers:         args.Env.RandomKeepers(nil, env.SupportsRotation(false)),
	},
		pulumi.Parent(comp),
		pulumi.IgnoreChanges(adminRandomPasswordIgnoreChanges),
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
		SkipAwait: pulumi.Bool(true),
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

func (c *Container) Scheme() pulumi.StringOutput { return c.scheme }
func (c *Container) Host() pulumi.StringOutput   { return c.host }
func (c *Container) Port() pulumi.StringOutput   { return c.port }
func (c *Container) Org() pulumi.StringOutput    { return c.org }
func (c *Container) Bucket() pulumi.StringOutput { return c.bucket }
func (c *Container) User() pulumi.StringOutput   { return c.user }
func (c *Container) Pass() pulumi.StringOutput   { return c.pass }
func (c *Container) Token() pulumi.StringOutput  { return c.token }
