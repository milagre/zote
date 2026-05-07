// Package digitalocean: mysql YAML cloud.digitalocean (Spec), Primary and Replicas (shared by Spec and Args), and the managed MySQL stack. Cluster is Pulumi-protected.
package digitalocean

import (
	"fmt"
	"strings"

	do "github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/cloud"
	"github.com/milagre/zote/pulumi/tokens"
)

var typeToken = tokens.Token("database", "MysqlDigitalocean")

var sqlMode = strings.Join([]string{
	"ANSI_QUOTES",
	"ERROR_FOR_DIVISION_BY_ZERO",
	"IGNORE_SPACE",
	"NO_ENGINE_SUBSTITUTION",
	"NO_ZERO_DATE",
	"NO_ZERO_IN_DATE",
	"ONLY_FULL_GROUP_BY",
	"PIPES_AS_CONCAT",
	"REAL_AS_FLOAT",
}, ",")

type Primary struct {
	Class string
}

type Replicas struct {
	Num   int
	Class string
}

// Args wires runtime identity, the multi-provider Cloud, and the YAML
// DigitalOcean spec under config.cloud.digitalocean.
type Args struct {
	Namespace string
	Name      string
	Database  string
	Version   string
	Cloud     cloud.Cloud

	// Config is mysql.Config.Cloud.DigitalOcean (YAML); must be non-nil.
	Config *Spec
}

type Digitalocean struct {
	pulumi.ResourceState

	username pulumi.StringOutput
	hostname pulumi.StringOutput
	port     pulumi.StringOutput
	password pulumi.StringOutput
}

// New resolves the DO database handle via Cloud.DigitalOcean.ForDatabase.
func New(ctx *pulumi.Context, parentName string, args *Args, opts ...pulumi.ResourceOption) (*Digitalocean, error) {
	if args == nil {
		return nil, fmt.Errorf("%s: args is required", typeToken)
	}
	if err := args.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	comp := &Digitalocean{}
	if err := ctx.RegisterComponentResource(typeToken, parentName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	dbCloud := args.Cloud.DigitalOcean.ForDatabase()
	vpcIDOut := dbCloud.VPCID().ToStringOutput().ToStringPtrOutput()
	projectIDOut := dbCloud.ProjectID().ToStringOutput().ToStringPtrOutput()

	vpc := do.LookupVpcOutput(ctx, do.LookupVpcOutputArgs{
		Id: vpcIDOut,
	})
	project := do.LookupProjectOutput(ctx, do.LookupProjectOutputArgs{
		Id: projectIDOut,
	})

	// The cluster's human-readable name is `<vpc>-<namespace>-<name>`.
	// The namespace is in the middle because DigitalOcean cluster
	// names are unique per-project: a bare `<vpc>-<name>` collides as
	// soon as two Kubernetes namespaces each ask for an instance of
	// the same logical name. vpc.Name() is an Output because the VPC
	// may only be resolved at apply time (see package doc on
	// zero-state construction); the namespace and instance-name
	// suffix are static and close over the Apply directly.
	//
	// IgnoreChanges on "name" is applied at the resource-opt level
	// below so pre-existing clusters whose historical names omit the
	// namespace segment are not renamed into compliance; the new
	// convention only governs greenfield deploys. A DigitalOcean
	// cluster rename would either be refused by the API or replace
	// the cluster (losing data), neither of which is acceptable.
	instanceName := args.Name
	namespace := args.Namespace
	clusterName := vpc.Name().ApplyT(func(vpcName string) string {
		return fmt.Sprintf("%s-%s-%s", vpcName, namespace, instanceName)
	}).(pulumi.StringOutput)

	cluster, err := do.NewDatabaseCluster(ctx, parentName, &do.DatabaseClusterArgs{
		Name:               clusterName,
		Engine:             pulumi.String("mysql"),
		Version:            pulumi.String(args.Version),
		Size:               pulumi.String(args.Config.Primary.Class),
		Region:             vpc.Region(),
		NodeCount:          pulumi.Int(1),
		PrivateNetworkUuid: vpc.Id(),
		ProjectId:          project.Id(),
	}, pulumi.Parent(comp), pulumi.Protect(true), pulumi.IgnoreChanges([]string{"name"}))
	if err != nil {
		return nil, fmt.Errorf("%s: database cluster: %w", typeToken, err)
	}

	if _, err := do.NewDatabaseFirewall(ctx, parentName+"-firewall", &do.DatabaseFirewallArgs{
		ClusterId: cluster.ID().ToStringOutput(),
		Rules: do.DatabaseFirewallRuleArray{
			&do.DatabaseFirewallRuleArgs{
				Type:  pulumi.String("ip_addr"),
				Value: vpc.IpRange(),
			},
		},
	}, pulumi.Parent(comp)); err != nil {
		return nil, fmt.Errorf("%s: firewall: %w", typeToken, err)
	}

	if _, err := do.NewDatabaseMysqlConfig(ctx, parentName+"-config", &do.DatabaseMysqlConfigArgs{
		ClusterId: cluster.ID().ToStringOutput(),
		SqlMode:   pulumi.String(sqlMode),
	}, pulumi.Parent(comp)); err != nil {
		return nil, fmt.Errorf("%s: mysql config: %w", typeToken, err)
	}

	// The schema is the unit of data durability on a managed mysql
	// instance: dropping it drops every row. Protect at the Pulumi
	// level so deletion is always an explicit, out-of-band action.
	if _, err := do.NewDatabaseDb(ctx, parentName+"-db", &do.DatabaseDbArgs{
		ClusterId: cluster.ID().ToStringOutput(),
		Name:      pulumi.String(args.Database),
	}, pulumi.Parent(comp), pulumi.Protect(true)); err != nil {
		return nil, fmt.Errorf("%s: database: %w", typeToken, err)
	}

	// Replicas follow the same `<vpc>-<namespace>-<name>-replica-<idx>`
	// convention as the primary cluster, for the same reason (unique
	// per project). They are marked protected + name-ignored for the
	// same reasons too: a replica rename is not a safe operation, and
	// a replica delete loses its independent read-only footprint
	// until DO re-syncs a fresh replica from the primary.
	//
	// The loop index (idx) is captured by value per-iteration so the
	// Apply closure below does not close over the loop-variant `i`,
	// which would otherwise emit every replica's name with the last
	// index. The static `instanceName` and `namespace` are closed over
	// directly because they do not change across iterations.
	replicaNum := 0
	replicaClass := ""
	if rp := args.Config.Replicas; rp != nil {
		replicaNum = rp.Num
		replicaClass = rp.Class
	}

	for i := 0; i < replicaNum; i++ {
		idx := i
		name := fmt.Sprintf("%s-replica-%d", parentName, idx)
		replicaName := vpc.Name().ApplyT(func(vpcName string) string {
			return fmt.Sprintf("%s-%s-%s-replica-%d", vpcName, namespace, instanceName, idx)
		}).(pulumi.StringOutput)

		if _, err := do.NewDatabaseReplica(ctx, name, &do.DatabaseReplicaArgs{
			ClusterId: cluster.ID().ToStringOutput(),
			Name:      replicaName,
			Size:      pulumi.String(replicaClass),
			Region:    vpc.Region(),
		}, pulumi.Parent(comp), pulumi.Protect(true), pulumi.IgnoreChanges([]string{"name"})); err != nil {
			return nil, fmt.Errorf("%s: replica %d: %w", typeToken, idx, err)
		}
	}

	comp.username = cluster.User
	comp.hostname = cluster.PrivateHost
	comp.port = cluster.Port.ApplyT(func(p int) string {
		return fmt.Sprintf("%d", p)
	}).(pulumi.StringOutput)
	comp.password = cluster.Password

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

// Username returns the default user name the managed cluster provisioned.
func (d *Digitalocean) Username() pulumi.StringOutput { return d.username }

// Hostname returns the private host the cluster is reachable at from
// inside the VPC.
func (d *Digitalocean) Hostname() pulumi.StringOutput { return d.hostname }

// Port returns the cluster's listening port as a string.
func (d *Digitalocean) Port() pulumi.StringOutput { return d.port }

// Password returns the default user's password generated by the managed
// service.
func (d *Digitalocean) Password() pulumi.StringOutput { return d.password }

func (a *Args) validate() error {
	if a.Name == "" {
		return fmt.Errorf("Name is required")
	}
	if a.Namespace == "" {
		return fmt.Errorf("Namespace is required")
	}
	if a.Database == "" {
		return fmt.Errorf("Database is required")
	}
	if a.Version == "" {
		return fmt.Errorf("Version is required")
	}
	if a.Config == nil {
		return fmt.Errorf("cloud.digitalocean config is required")
	}
	if err := a.Config.Validate(); err != nil {
		return fmt.Errorf("digitalocean config: %w", err)
	}
	if a.Cloud.DigitalOcean == nil {
		return fmt.Errorf("Cloud.DigitalOcean is required")
	}
	dbCloud := a.Cloud.DigitalOcean.ForDatabase()
	if dbCloud.VPCID() == nil {
		return fmt.Errorf("Cloud.DigitalOcean ForDatabase VPCID is nil")
	}
	if dbCloud.ProjectID() == nil {
		return fmt.Errorf("Cloud.DigitalOcean ForDatabase ProjectID is nil")
	}

	return nil
}
