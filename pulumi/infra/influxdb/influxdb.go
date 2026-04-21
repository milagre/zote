// Package influxdb provides a ComponentResource that deploys an influxdb
// instance and exposes its connection details via a ConfigMap and Secret
// in the target namespace.
//
// The component is backend-polymorphic: today only the container backend is
// implemented (via an in-cluster Helm chart), but the shape of Args leaves
// room for additional backends (e.g. managed cloud InfluxDB) to be added
// without changing the client-facing resources.
package influxdb

import (
	"fmt"
	"strings"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra/influxdb/internal/container"
	"github.com/milagre/zote/pulumi/tokens"
	"github.com/milagre/zote/pulumi/stringdata"
)

var typeToken = tokens.Token("infra", "Influxdb")

// Default admin organization and username, applied when the caller leaves
// Args.Organization / Args.User empty. Both are influxdb concepts rather
// than container-backend concepts, so the defaults live with the parent
// component and the backend always receives resolved values.
const (
	defaultOrg  = "influxdb"
	defaultUser = "admin"
)

// ContainerArgs is the caller-facing configuration for the in-cluster
// container backend. It is a type alias over the internal implementation
// type so callers can populate it without importing the internal package.
type ContainerArgs = container.Args

// Args configures a new Influxdb instance.
//
// Exactly one of the backend pointer fields (currently just Container) must
// be non-nil. Setting none or more than one is rejected at construction
// time so the choice of backend is explicit in the call site and so no
// in-cluster resources are created when a managed backend is selected.
type Args struct {
	// Env is the deploy environment (used to derive the ConfigMap/Secret
	// key prefix).
	Env env.Env
	// Namespace is the target Kubernetes namespace for all resources.
	Namespace string
	// Name is the logical influxdb instance name; used both as the release
	// name and as part of the ConfigMap/Secret key prefix.
	Name string
	// Organization is the influxdb admin organization (and default bucket
	// name). Empty falls back to defaultOrg.
	Organization string
	// User is the influxdb admin username. Empty falls back to defaultUser.
	User string

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

// Influxdb is the component resource. Connection details are not exposed
// directly; they live in the ConfigMap and Secret named by K8s and are
// consumed by mounting those resources into dependent workloads.
type Influxdb struct {
	pulumi.ResourceState

	K8s K8s
}

// New registers the Influxdb component and its chosen backend.
func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Influxdb, error) {
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

	comp := &Influxdb{}
	if err := ctx.RegisterComponentResource(typeToken, resourceName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	be, err := selectBackend(ctx, resourceName, args, comp)
	if err != nil {
		return nil, err
	}

	cfgPrefix := fmt.Sprintf("%s_INFLUXDB_%s", args.Env.Prefix, strings.ToUpper(args.Name))
	clientName := fmt.Sprintf("influxdb-%s", args.Name)
	labels := pulumi.StringMap{"app": pulumi.String(clientName)}

	cm, err := corev1.NewConfigMap(ctx, resourceName, &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(clientName),
			Namespace: pulumi.String(args.Namespace),
			Labels:    labels,
		},
		Data: pulumi.StringMap{
			cfgPrefix + "_SCHEME": be.Scheme(),
			cfgPrefix + "_HOST":   be.Host(),
			cfgPrefix + "_PORT":   be.Port(),
			cfgPrefix + "_ORG":    be.Org(),
			cfgPrefix + "_BUCKET": be.Bucket(),
			cfgPrefix + "_USER":   be.User(),
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
			Annotations: pulumi.StringMap{"pulumi.com/patchForce": pulumi.String("true")},
		},
		Type: pulumi.String("Opaque"),
		Data: stringdata.SecretData(map[string]pulumi.StringOutput{
			cfgPrefix + "_PASS":  be.Pass(),
			cfgPrefix + "_TOKEN": be.Token(),
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

// backend is the internal contract every backend implementation satisfies.
// Values flow directly into the shared ConfigMap/Secret, so everything is
// exposed as string outputs — scalar types (int ports) are stringified by
// the implementation.
type backend interface {
	Scheme() pulumi.StringOutput
	Host() pulumi.StringOutput
	Port() pulumi.StringOutput
	Org() pulumi.StringOutput
	Bucket() pulumi.StringOutput
	User() pulumi.StringOutput
	Pass() pulumi.StringOutput
	Token() pulumi.StringOutput
}

func selectBackend(ctx *pulumi.Context, name string, args *Args, parent pulumi.Resource) (backend, error) {
	org := args.Organization
	if org == "" {
		org = defaultOrg
	}
	user := args.User
	if user == "" {
		user = defaultUser
	}

	cArgs := *args.Container
	cArgs.Env = args.Env
	cArgs.Namespace = args.Namespace
	cArgs.Name = args.Name
	cArgs.Organization = org
	cArgs.User = user
	c, err := container.New(ctx, name, &cArgs, parent)
	if err != nil {
		return nil, fmt.Errorf("%s: container backend: %w", typeToken, err)
	}

	return c, nil
}
