// Package container is the in-cluster implementation of the redis backend
// interface defined in the parent redis package. It provisions a redis
// cluster StatefulSet, headless service, config/script ConfigMaps, and a
// bootstrap Job that forms the cluster after the pods are ready.
package container

import (
	"fmt"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	batchv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/batch/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/tokens"
	"github.com/milagre/zote/pulumi/profile"
)

const (
	clientPort  = 6379
	clusterPort = 16379
)

var typeToken = tokens.Token("infra", "RedisContainer")

// Args is the caller-facing configuration for a container-backed redis.
type Args struct {
	// Namespace is the target Kubernetes namespace.
	Namespace string
	// Name is the instance name (release name "redis-<Name>").
	Name string
	// Version is the redis container image tag.
	Version string
	// Profile is the validated resource profile (CPU, memory). Storage is
	// sized from mem.max * 1.1.
	Profile profile.Profile
	// Shards is the number of master shards to form.
	Shards int
	// Replicas is the number of replicas per master. Total pods in the
	// StatefulSet equal Shards * (Replicas + 1).
	Replicas int
}

// Container is the container-backed implementation of the redis backend.
type Container struct {
	pulumi.ResourceState

	StatefulSet *appsv1.StatefulSet
	Service     *corev1.Service
	ConfigMap   *corev1.ConfigMap
	Scripts     *corev1.ConfigMap
	Cluster     *batchv1.Job

	hostname pulumi.StringOutput
	port     pulumi.StringOutput
}

// New registers the container backend and every resource it owns.
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
	if args.Shards <= 0 {
		return nil, fmt.Errorf("%s: Shards must be > 0", typeToken)
	}
	if args.Replicas < 0 {
		return nil, fmt.Errorf("%s: Replicas must be >= 0", typeToken)
	}

	comp := &Container{}
	if err := ctx.RegisterComponentResource(typeToken, parentName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	releaseName := fmt.Sprintf("redis-%s", args.Name)

	svc, err := registerHeadlessService(ctx, parentName, comp, args.Namespace, releaseName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	cfg, scripts, err := registerConfig(ctx, parentName, comp, args.Namespace, releaseName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	sts, err := registerStatefulSet(ctx, parentName, comp, args, releaseName, svc, cfg, scripts)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	job, err := registerBootstrapJob(ctx, parentName, comp, args, releaseName, scripts, sts)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	comp.StatefulSet = sts
	comp.Service = svc
	comp.ConfigMap = cfg
	comp.Scripts = scripts
	comp.Cluster = job

	comp.hostname = pulumi.String(fmt.Sprintf("%s.%s.svc.cluster.local", releaseName, args.Namespace)).ToStringOutput()
	comp.port = pulumi.Sprintf("%d", clientPort).ToStringOutput()

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

// Hostname returns the headless service FQDN.
func (c *Container) Hostname() pulumi.StringOutput { return c.hostname }

// Port returns the client port as a string.
func (c *Container) Port() pulumi.StringOutput { return c.port }
