// Package timescaledb provides a ComponentResource that deploys a
// timescaledb instance and exposes its connection details via a ConfigMap
// and Secret in the target namespace.
//
// The component is backend-polymorphic: today only the container backend is
// implemented (as an in-cluster StatefulSet), but the shape of Args leaves
// room for additional backends (managed cloud postgres, etc.) to be added
// without changing the client-facing resources.
package timescaledb

import (
	"fmt"
	"strings"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra/timescaledb/internal/container"
	"github.com/milagre/zote/pulumi/stringdata"
	"github.com/milagre/zote/pulumi/tokens"
)

var typeToken = tokens.Token("infra", "Timescaledb")

// Default admin user and database, applied when the caller leaves
// Args.User / Args.Database empty. Both are postgres/timescaledb concepts
// rather than container-backend concepts, so the defaults live with the
// parent component and the backend always receives resolved values.
const (
	defaultUser     = "timescaledb"
	defaultDatabase = "timescaledb"
)

// ContainerArgs is the caller-facing configuration for the in-cluster
// container backend. It is a type alias over the internal implementation
// type so callers can populate it without importing the internal package.
type ContainerArgs = container.Args

// Args configures a new Timescaledb instance. Exactly one backend pointer
// must be non-nil (currently just Container).
type Args struct {
	Env       env.Env
	Namespace string
	Name      string
	// User is the postgres admin username. Empty falls back to
	// defaultUser.
	User string
	// Database is the initial postgres database name. Empty falls back
	// to defaultDatabase.
	Database string

	// Container selects the in-cluster container backend. Mutually
	// exclusive with future cloud-backend pointers.
	Container *ContainerArgs
}

// K8s holds the names of the Kubernetes resources the component creates
// in the target namespace. Grouping the names under a struct (rather than
// flattening them onto the component) makes it obvious at the call site
// that the strings are in-cluster resource names, not arbitrary
// configuration values.
type K8s struct {
	ConfigMap pulumi.StringOutput
	Secret    pulumi.StringOutput
}

// Timescaledb is the component resource. Connection details are not
// exposed directly; they live in the ConfigMap and Secret named by K8s
// and are consumed by mounting those resources into dependent workloads.
type Timescaledb struct {
	pulumi.ResourceState

	K8s K8s
}

// New registers the Timescaledb component and its chosen backend.
func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Timescaledb, error) {
	if args == nil {
		return nil, fmt.Errorf("%s: args is required", typeToken)
	}
	if args.Name == "" {
		return nil, fmt.Errorf("%s: Name is required", typeToken)
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("%s: Namespace is required", typeToken)
	}
	if err := args.Env.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}
	if args.Container == nil {
		return nil, fmt.Errorf("%s: a backend is required (currently only Container is implemented)", typeToken)
	}

	resourceName := tokens.Qualify(args.Namespace, name)

	comp := &Timescaledb{}
	if err := ctx.RegisterComponentResource(typeToken, resourceName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	be, err := selectBackend(ctx, resourceName, args, comp)
	if err != nil {
		return nil, err
	}

	cfgPrefix := fmt.Sprintf("%s_TIMESCALEDB_%s", args.Env.Prefix, strings.ToUpper(args.Name))
	clientName := fmt.Sprintf("timescaledb-%s", args.Name)
	labels := pulumi.StringMap{"app": pulumi.String(clientName)}

	patchForce := pulumi.StringMap{"pulumi.com/patchForce": pulumi.String("true")}

	cm, err := corev1.NewConfigMap(ctx, resourceName, &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(clientName),
			Namespace:   pulumi.String(args.Namespace),
			Labels:      labels,
			Annotations: patchForce,
		},
		Data: pulumi.StringMap{
			cfgPrefix + "_SCHEME":   be.Scheme(),
			cfgPrefix + "_HOST":     be.Host(),
			cfgPrefix + "_PORT":     be.Port(),
			cfgPrefix + "_USER":     be.User(),
			cfgPrefix + "_DATABASE": be.Database(),
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: client configmap: %w", typeToken, err)
	}

	sec, err := corev1.NewSecret(ctx, resourceName, &corev1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(clientName),
			Namespace:   pulumi.String(args.Namespace),
			Labels:      labels,
			Annotations: patchForce,
		},
		Type: pulumi.String("Opaque"),
		Data: stringdata.SecretData(map[string]pulumi.StringOutput{
			cfgPrefix + "_PASSWORD": be.Pass(),
		}),
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: client secret: %w", typeToken, err)
	}

	comp.K8s.ConfigMap = cm.Metadata.Name().Elem()
	comp.K8s.Secret = sec.Metadata.Name().Elem()

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"k8s": pulumi.Map{
			"configMap": comp.K8s.ConfigMap,
			"secret":    comp.K8s.Secret,
		},
	}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

type backend interface {
	Scheme() pulumi.StringOutput
	Host() pulumi.StringOutput
	Port() pulumi.StringOutput
	User() pulumi.StringOutput
	Pass() pulumi.StringOutput
	Database() pulumi.StringOutput
}

func selectBackend(ctx *pulumi.Context, name string, args *Args, parent pulumi.Resource) (backend, error) {
	user := args.User
	if user == "" {
		user = defaultUser
	}
	database := args.Database
	if database == "" {
		database = defaultDatabase
	}

	cArgs := *args.Container
	cArgs.Env = args.Env
	cArgs.Namespace = args.Namespace
	cArgs.Name = args.Name
	cArgs.User = user
	cArgs.Database = database
	c, err := container.New(ctx, name, &cArgs, pulumi.Parent(parent))
	if err != nil {
		return nil, fmt.Errorf("%s: container backend: %w", typeToken, err)
	}

	return c, nil
}
