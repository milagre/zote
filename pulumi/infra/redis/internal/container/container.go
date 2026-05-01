// Package container is Redis cluster in-cluster (StatefulSet, headless svc, configmaps, bootstrap Job).
package container

import (
	"fmt"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	batchv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/batch/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/profile"
	"github.com/milagre/zote/pulumi/tokens"
)

const (
	clientPort  = 6379
	clusterPort = 16379
)

var typeToken = tokens.Token("infra", "RedisContainer")

type Args struct {
	Namespace string
	Name      string
	Version   string
	Profile   profile.Profile
	Shards    int
	Replicas  int // StatefulSet size = Shards * (Replicas + 1)
}

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

func (c *Container) Hostname() pulumi.StringOutput { return c.hostname }
func (c *Container) Port() pulumi.StringOutput     { return c.port }
