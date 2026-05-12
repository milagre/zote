// Package loki deploys Grafana Loki with object storage (S3-compatible).
// Topology (SingleBinary vs SimpleScalable) follows [Config.Monolithic].
package loki

import (
	"fmt"
	"net/url"

	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/endpoint"
	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra/objectstorage"
	"github.com/milagre/zote/pulumi/profile"
	"github.com/milagre/zote/pulumi/tokens"
)

const (
	chart          = "loki"
	repository     = "https://grafana.github.io/helm-charts"
	defaultVersion = "7.0.0"
)

// Loki Helm defaults auth_enabled true, which requires a X-Scope-OrgID on every request.
const authEnabled = false

var typeToken = tokens.Token("infra", "Loki")

// Args bundles the YAML-decoded Config with the multi-provider Cloud
// container. Cloud is consulted only when Config.Cloud is populated;
// the provider field matching Config.Cloud must be non-nil.
type Args struct {
	Env       env.Env
	Namespace string

	Config Config

	ObjectStorage objectstorage.ObjectStorage
}

type Loki struct {
	pulumi.ResourceState

	Gateway url.URL
	Push    url.URL

	// PushURL and GatewayURL mirror [Push]/[Gateway] but depend on the Helm release ID,
	// so downstream configs that embed them wait until Loki is registered.
	PushURL    pulumi.StringOutput
	GatewayURL pulumi.StringOutput

	Deps []pulumi.Resource // pass to DependsOn
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Loki, error) {
	if args == nil {
		return nil, fmt.Errorf("%s: args is required", typeToken)
	}
	if err := args.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	resourceName := tokens.Qualify(args.Namespace, name)

	comp := &Loki{}
	if err := ctx.RegisterComponentResource(typeToken, resourceName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	values, deps, err := args.values()
	if err != nil {
		return nil, fmt.Errorf("%s: values: %w", typeToken, err)
	}

	version := pulumi.String(defaultVersion).ToStringPtrOutput()
	if args.Config.Version != "" {
		version = pulumi.String(args.Config.Version).ToStringPtrOutput()
	}

	relOpts := []pulumi.ResourceOption{pulumi.Parent(comp)}
	if len(deps) > 0 {
		relOpts = append(relOpts, pulumi.DependsOn(deps))
	}

	rel, err := helmv3.NewRelease(ctx, resourceName, &helmv3.ReleaseArgs{
		Chart:     pulumi.String(chart),
		Name:      pulumi.String(name).ToStringPtrOutput(),
		Namespace: pulumi.String(args.Namespace).ToStringPtrOutput(),
		Version:   version,
		RepositoryOpts: &helmv3.RepositoryOptsArgs{
			Repo: pulumi.String(repository).ToStringPtrOutput(),
		},
		Values: values,
	}, relOpts...)
	if err != nil {
		return nil, fmt.Errorf("%s: helm release: %w", typeToken, err)
	}

	gwHost := fmt.Sprintf("%s-gateway.%s.svc.cluster.local", name, args.Namespace)
	comp.Gateway = endpoint.HTTP(gwHost, "80", "/")
	comp.Push = endpoint.HTTP(gwHost, "80", "/loki/api/v1/push")
	comp.Deps = []pulumi.Resource{rel}

	ps := comp.Push.String()
	gw := comp.Gateway.String()
	comp.PushURL = rel.ID().ApplyT(func(pulumi.ID) string { return ps }).(pulumi.StringOutput)
	comp.GatewayURL = rel.ID().ApplyT(func(pulumi.ID) string { return gw }).(pulumi.StringOutput)

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"pushUrl":    comp.PushURL,
		"gatewayUrl": comp.GatewayURL,
	}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

func (a *Args) values() (pulumi.Map, []pulumi.Resource, error) {
	s3Bucket, err := a.ObjectStorage.ProvisionedBucket(a.Config.Bucket)
	if err != nil {
		return nil, nil, fmt.Errorf("bucket: %w", err)
	}

	prof, err := profile.New(a.Config.Profile)
	if err != nil {
		return nil, nil, fmt.Errorf("profile: %w", err)
	}
	resources := pulumi.Map{
		"requests": pulumi.Map{
			"cpu":    pulumi.String(prof.MinCoresMilli()),
			"memory": pulumi.String(prof.MinMemMiB()),
		},
		"limits": pulumi.Map{
			"cpu":    pulumi.String(prof.MaxCoresMilli()),
			"memory": pulumi.String(prof.MaxMemMiB()),
		},
	}

	var mode string
	var (
		writeN   int
		readN    int
		backendN int
		singleN  int
	)
	if a.Config.Monolithic {
		mode = "SingleBinary"
		writeN = 0
		readN = 0
		backendN = 0
		singleN = 1
	} else {
		mode = "SimpleScalable"
		writeN = 2
		readN = 2
		backendN = 2
		singleN = 0
	}

	lokiBlock := pulumi.Map{
		"auth_enabled": pulumi.Bool(authEnabled),
		"storage": pulumi.Map{
			"type": pulumi.String("s3"),
			"bucketNames": pulumi.Map{
				"chunks": pulumi.String(s3Bucket),
				"ruler":  pulumi.String(s3Bucket),
				"admin":  pulumi.String(s3Bucket),
			},
			"s3": pulumi.Map{
				"endpoint":        a.ObjectStorage.S3.Addr(),
				"accessKeyId":     a.ObjectStorage.Creds.AccessKey,
				"secretAccessKey": a.ObjectStorage.Creds.SecretKey,
				"insecure":        a.ObjectStorage.Insecure,
				// For emulators we typically want path-style; Spaces also supports it.
				"s3ForcePathStyle": pulumi.Bool(true),
			},
		},
		"schemaConfig": pulumi.Map{
			"configs": pulumi.Array{
				pulumi.Map{
					"from":         pulumi.String("2024-04-01"),
					"store":        pulumi.String("tsdb"),
					"object_store": pulumi.String("s3"),
					"schema":       pulumi.String("v13"),
					"index": pulumi.Map{
						"prefix": pulumi.String("index_"),
						"period": pulumi.String("24h"),
					},
				},
			},
		},
	}
	// Helm default replication_factor is 3; unhealthy ring with fewer ingestors yields failed queries (often nginx 502 on the gateway).
	writeReplicas := writeN
	if a.Config.Monolithic {
		writeReplicas = singleN
	}
	lokiBlock["commonConfig"] = pulumi.Map{
		"replication_factor": pulumi.Int(replicationFactor(writeReplicas)),
	}

	values := pulumi.Map{
		"deploymentMode": pulumi.String(mode),
		"loki":           lokiBlock,
		"write": pulumi.Map{
			"replicas":  pulumi.Int(writeN),
			"resources": resources,
		},
		"read": pulumi.Map{
			"replicas":  pulumi.Int(readN),
			"resources": resources,
		},
		"backend": pulumi.Map{
			"replicas":  pulumi.Int(backendN),
			"resources": resources,
		},
		"singleBinary": pulumi.Map{
			"replicas":  pulumi.Int(singleN),
			"resources": resources,
		},
		"gateway":      pulumi.Map{"enabled": pulumi.Bool(true)},
		"chunksCache":  pulumi.Map{"enabled": pulumi.Bool(false)},
		"resultsCache": pulumi.Map{"enabled": pulumi.Bool(false)},
		"test":         pulumi.Map{"enabled": pulumi.Bool(false)},
		"lokiCanary":   pulumi.Map{"enabled": pulumi.Bool(false)},
		"monitoring": pulumi.Map{
			"selfMonitoring": pulumi.Map{"enabled": pulumi.Bool(false)},
		},
	}

	if a.Config.Monolithic {
		for k, v := range distributedMicroserviceZeros() {
			values[k] = v
		}
		// Headless loki-memberlist has no DNS records until at least one endpoint exists; without this,
		// startup memberlist join hits NXDOMAIN until readiness (grafana/loki#7907).
		values["memberlist"] = pulumi.Map{
			"service": pulumi.Map{
				"publishNotReadyAddresses": pulumi.Bool(true),
			},
		}
	}

	return values, a.ObjectStorage.Deps, nil
}

func replicationFactor(writeReplicas int) int {
	if writeReplicas < 1 {
		return 1
	}

	return writeReplicas
}

// helmTopology selects Grafana chart deploymentMode and replica counts (see production/helm/loki/single-binary-values.yaml).
func helmTopology(monolithic bool) (deploymentMode string, writeN, readN, backendN, singleN int) {
	if monolithic {
		return "SingleBinary", 0, 0, 0, 1
	}

	return "SimpleScalable", 2, 2, 2, 0
}

// distributedMicroserviceZeros disables distributed-mode targets when using SingleBinary (chart defaults may be non-zero).
func distributedMicroserviceZeros() pulumi.Map {
	z := pulumi.Int(0)
	return pulumi.Map{
		"ingester":       pulumi.Map{"replicas": z},
		"querier":        pulumi.Map{"replicas": z},
		"queryFrontend":  pulumi.Map{"replicas": z},
		"queryScheduler": pulumi.Map{"replicas": z},
		"distributor":    pulumi.Map{"replicas": z},
		"compactor":      pulumi.Map{"replicas": z},
		"indexGateway":   pulumi.Map{"replicas": z},
		"bloomCompactor": pulumi.Map{"replicas": z},
		"bloomGateway":   pulumi.Map{"replicas": z},
	}
}

func (a *Args) validate() error {
	if a.Namespace == "" {
		return fmt.Errorf("Namespace is required")
	}
	if err := a.Env.Validate(); err != nil {
		return fmt.Errorf("invalid env: %w", err)
	}
	if err := a.Config.Validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if !bucketIn(a.Config.Bucket, a.ObjectStorage.Buckets) {
		return fmt.Errorf("ObjectStorage does not contain bucket %q", a.Config.Bucket)
	}

	return nil
}

func bucketIn(bucket string, buckets map[string]string) bool {
	if buckets == nil {
		return false
	}

	_, ok := buckets[bucket]

	return ok
}
