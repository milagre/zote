// Package digitalocean is the DigitalOcean-managed implementation of
// the mysql backend interface defined in the parent mysql package. It
// provisions a managed DatabaseCluster (engine "mysql"), a firewall
// restricting access to the cluster's VPC, a seeded database, and any
// requested read-replicas.
//
// The cluster is marked protected at the Pulumi level: destroying a
// database through Pulumi is always an explicit, out-of-band action.
package digitalocean

import (
	"fmt"
	"strings"

	do "github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	dbdo "github.com/milagre/zote/pulumi/database/digitalocean"
	"github.com/milagre/zote/pulumi/tokens"
)

var typeToken = tokens.Token("database", "MysqlDigitalocean")

// sqlMode is the set of MySQL modes the managed config forces on the
// cluster. Mirrors the strictness the container backend inherits from
// modern mysql defaults so migrations between backends don't trip over
// previously-tolerated queries.
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

// Primary configures the single writer node.
type Primary struct {
	// Class is the DigitalOcean size slug (e.g. "db-s-1vcpu-1gb").
	Class string
}

// Replicas configures the read-replica fleet. Num=0 skips all replicas.
type Replicas struct {
	Num   int
	Class string
}

// Args is the caller-facing configuration for a DigitalOcean-managed
// mysql instance. The parent mysql component fills in the instance
// identity (Name/Namespace); the caller supplies engine settings,
// sizing, and a per-instance Cloud handle (typically obtained via
// cloud/digitalocean.Cloud.ForDatabase) that resolves the VPC and
// project the cluster should live in.
type Args struct {
	// Namespace is the Kubernetes namespace this instance serves. It
	// is baked into the cluster's human-readable name so two mysql
	// instances that share a Name but live in different namespaces
	// produce distinct cluster names — DigitalOcean cluster names are
	// unique per project, not per VPC/subnet/label, so namespacing is
	// the only free disambiguator.
	Namespace string
	// Name is the logical mysql instance name; also used to derive the
	// cluster's human-readable name via VPCName + "-" + Namespace +
	// "-" + Name.
	Name string
	// Database is the schema name created inside the cluster.
	Database string
	// Version is the MySQL version slug accepted by the DO API (e.g. "8").
	Version string
	// Cloud resolves the VPC/project this specific database instance
	// belongs to. Two databases with different VPCs are expected to
	// receive two different Cloud values.
	Cloud dbdo.Cloud
	// Primary is the writer node sizing.
	Primary Primary
	// Replicas is the read-replica fleet sizing.
	Replicas Replicas
}

// Digitalocean provisions a managed MySQL cluster and exposes the
// connection details the parent component wires into its shared
// ConfigMap/Secret.
type Digitalocean struct {
	pulumi.ResourceState

	username pulumi.StringOutput
	hostname pulumi.StringOutput
	port     pulumi.StringOutput
	password pulumi.StringOutput
}

// New registers the DigitalOcean backend as a child component of the
// parent mysql facade.
//
// The VPC + project IDs this instance should live in are consumed as
// pulumi.StringInput from args.Cloud, which means they may be
// unresolved Outputs of cloud resources that the same Pulumi program
// creates earlier in the stack. Every piece of downstream work that
// depends on those IDs is therefore issued through the *Output
// variants of the DigitalOcean data sources (LookupVpcOutput,
// LookupProjectOutput), which accept StringInputs and return outputs
// Pulumi resolves in dependency order at apply time. No call in this
// function reads a Go-level string from the Cloud handle.
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

	vpcIDOut := args.Cloud.VPCID().ToStringOutput().ToStringPtrOutput()
	projectIDOut := args.Cloud.ProjectID().ToStringOutput().ToStringPtrOutput()

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
		Size:               pulumi.String(args.Primary.Class),
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
	for i := 0; i < args.Replicas.Num; i++ {
		idx := i
		name := fmt.Sprintf("%s-replica-%d", parentName, idx)
		replicaName := vpc.Name().ApplyT(func(vpcName string) string {
			return fmt.Sprintf("%s-%s-%s-replica-%d", vpcName, namespace, instanceName, idx)
		}).(pulumi.StringOutput)

		if _, err := do.NewDatabaseReplica(ctx, name, &do.DatabaseReplicaArgs{
			ClusterId: cluster.ID().ToStringOutput(),
			Name:      replicaName,
			Size:      pulumi.String(args.Replicas.Class),
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
	// Cloud is the only input that can meaningfully fail the
	// type-level nil check here; VPCID() / ProjectID() return
	// pulumi.StringInput which is an interface and may legitimately
	// carry an unresolved pulumi.Output. An empty-string check would
	// require blocking on the Output's value, which is exactly what the
	// pulumi.StringInput signature exists to avoid. A downstream
	// LookupVpc / LookupProject failure surfaces a clearer error at
	// apply time if the ID ultimately resolves to something invalid.
	if a.Cloud == nil {
		return fmt.Errorf("Cloud is required")
	}
	if a.Cloud.VPCID() == nil {
		return fmt.Errorf("Cloud.VPCID is nil")
	}
	if a.Cloud.ProjectID() == nil {
		return fmt.Errorf("Cloud.ProjectID is nil")
	}
	if a.Primary.Class == "" {
		return fmt.Errorf("Primary.Class is required")
	}
	if a.Replicas.Num > 0 && a.Replicas.Class == "" {
		return fmt.Errorf("Replicas.Class is required when Replicas.Num > 0")
	}

	return nil
}
