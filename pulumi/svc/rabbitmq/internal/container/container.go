// Package container is in-cluster RabbitMQ (StatefulSet, services, RBAC, k8s peer discovery).
package container

import (
	"fmt"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/util/profile"
	"github.com/milagre/zote/pulumi/util/tokens"
)

var typeToken = tokens.Token("svc", "RabbitmqContainer")

var randomPasswordIgnoredArgs = []string{
	"length", "special", "upper", "lower", "numeric",
	"minLower", "minUpper", "minNumeric", "minSpecial", "overrideSpecial",
}

type User struct {
	Name string
	Tags []string
}

type Vhost struct {
	Name  string
	Users []string
}

type Setup struct {
	Users  []User
	Vhosts []Vhost
}

type Args struct {
	Env       env.Env
	Namespace string
	Name      string
	Version   string
	Profile   profile.Profile
	Setup     Setup
}

type Container struct {
	pulumi.ResourceState

	StatefulSet     *appsv1.StatefulSet
	ClientService   *corev1.Service
	HeadlessService *corev1.Service

	hostname      pulumi.StringOutput
	port          pulumi.StringOutput
	adminHostname pulumi.StringOutput
	adminPort     pulumi.StringOutput

	passwords       map[string]pulumi.StringOutput
	monitorPassword pulumi.StringOutput
	users           []string
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
	if err := args.Env.Validate(); err != nil {
		return nil, fmt.Errorf("%s: env: %w", typeToken, err)
	}

	comp := &Container{}
	if err := ctx.RegisterComponentResource(typeToken, parentName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	releaseName := fmt.Sprintf("rabbitmq-%s", args.Name)
	// Synthetic admin and monitor users are always present regardless of caller
	// setup: admin so operators always have a management-scope login, monitor so
	// autoscalers have a least-privilege queue-depth reader. Both get per-vhost
	// permission rows (see definitionsJSON): admin full access, monitor read-only
	// (which is required for KEDA to reach vhost-scoped queue endpoints).
	allUsers := make([]User, 0, len(args.Setup.Users)+2)
	allUsers = append(allUsers, args.Setup.Users...)
	allUsers = append(allUsers, User{Name: adminUser, Tags: []string{"administrator", "management"}})
	allUsers = append(allUsers, User{Name: monitorUser, Tags: []string{"monitoring"}})

	userNames := make([]string, 0, len(allUsers))
	for _, u := range allUsers {
		userNames = append(userNames, u.Name)
	}

	creds, err := registerCreds(ctx, parentName, comp, userNames, args.Env)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	defaultPassword, err := random.NewRandomPassword(ctx, parentName+"-default-password", &random.RandomPasswordArgs{
		Length:  pulumi.Int(32),
		Numeric: pulumi.Bool(true),
		Upper:   pulumi.Bool(true),
		Lower:   pulumi.Bool(true),
		Special: pulumi.Bool(false),
		Keepers: args.Env.RandomKeepers(nil),
	},
		pulumi.Parent(comp),
		pulumi.IgnoreChanges(randomPasswordIgnoredArgs),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: default password: %w", typeToken, err)
	}
	erlangCookie, err := random.NewRandomPassword(ctx, parentName+"-erlang-cookie", &random.RandomPasswordArgs{
		Length:  pulumi.Int(32),
		Numeric: pulumi.Bool(true),
		Upper:   pulumi.Bool(true),
		Lower:   pulumi.Bool(true),
		Special: pulumi.Bool(false),
		Keepers: args.Env.RandomKeepers(nil),
	},
		pulumi.Parent(comp),
		pulumi.IgnoreChanges(randomPasswordIgnoredArgs),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: erlang cookie: %w", typeToken, err)
	}

	sa, err := registerRBAC(ctx, parentName, comp, args.Namespace, releaseName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	cfgCM, cfgSecret, err := configResources(
		ctx, parentName, comp,
		args.Namespace, releaseName,
		creds, allUsers, args.Setup.Vhosts,
		erlangCookie, defaultPassword,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	sts, client, headless, err := registerWorkload(
		ctx, parentName, comp,
		args.Namespace, args.Name, releaseName, args.Version,
		args.Profile, cfgCM, cfgSecret, sa,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	comp.StatefulSet = sts
	comp.ClientService = client
	comp.HeadlessService = headless

	fqdn := pulumi.String(fmt.Sprintf("%s.%s.svc.cluster.local", releaseName, args.Namespace)).ToStringOutput()

	comp.hostname = fqdn
	comp.port = pulumi.Sprintf("%d", portAMQP).ToStringOutput()
	comp.adminHostname = fqdn
	comp.adminPort = pulumi.Sprintf("%d", portManagement).ToStringOutput()

	// Expose only the caller-supplied users (admin is internal) — same as
	// the legacy module's `passwords` output.
	comp.passwords = make(map[string]pulumi.StringOutput, len(args.Setup.Users))
	for _, u := range args.Setup.Users {
		comp.passwords[u.Name] = creds[u.Name].password.Result
	}
	comp.monitorPassword = creds[monitorUser].password.Result
	comp.users = userNames

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

func (c *Container) Hostname() pulumi.StringOutput             { return c.hostname }
func (c *Container) Port() pulumi.StringOutput                 { return c.port }
func (c *Container) AdminHostname() pulumi.StringOutput        { return c.adminHostname }
func (c *Container) AdminPort() pulumi.StringOutput            { return c.adminPort }
func (c *Container) Passwords() map[string]pulumi.StringOutput { return c.passwords }
func (c *Container) MonitorUser() string                       { return monitorUser }
func (c *Container) MonitorPassword() pulumi.StringOutput      { return c.monitorPassword }
