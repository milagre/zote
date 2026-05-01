// Package influxdb deploys influxdb and publishes connection details in a
// ConfigMap and Secret. Only the container (Helm) backend is implemented; Args
// is shaped for additional backends later.
package influxdb

import (
	"fmt"
	"strings"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra/influxdb/internal/container"
	"github.com/milagre/zote/pulumi/stringdata"
	"github.com/milagre/zote/pulumi/tokens"
)

var typeToken = tokens.Token("infra", "Influxdb")

const (
	defaultOrg  = "influxdb"
	defaultUser = "admin"
)

// ContainerArgs is an alias for [container.Args] (avoids importing internal).
type ContainerArgs = container.Args

// Args configures Influxdb. Container is required today. Env supplies WM_* key prefix;
// empty Organization/User use defaults.
type Args struct {
	Env          env.Env
	Namespace    string
	Name         string
	Organization string
	User         string
	Container    *ContainerArgs
}

// K8s names the ConfigMap and Secret created in the target namespace.
type K8s struct {
	ConfigMap pulumi.StringOutput
	Secret    pulumi.StringOutput
}

// Influxdb is the component resource; connection details are in K8s only.
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
