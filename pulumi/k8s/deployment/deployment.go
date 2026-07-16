// Package deployment is the HTTP vs proc workload facade; hostnames are unified here.
package deployment

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/k8s/deployment/http"
	"github.com/milagre/zote/pulumi/k8s/deployment/internal/dashboard"
	"github.com/milagre/zote/pulumi/k8s/deployment/internal/scaledobject"
	"github.com/milagre/zote/pulumi/k8s/deployment/proc"
	"github.com/milagre/zote/pulumi/k8s/internal/podspec"
	"github.com/milagre/zote/pulumi/util/profile"
	"github.com/milagre/zote/pulumi/util/tokens"
)

var typeToken = tokens.Token("k8s", "Deployment")

type (
	Conf     = podspec.Conf
	Files    = podspec.Files
	HTTPMode = http.Options

	// Autoscale configures KEDA-driven scaling for a workload. Replica bounds
	// come from Profile.Num
	Autoscale          = scaledobject.Spec
	QueueTrigger       = scaledobject.QueueTrigger
	UtilizationTrigger = scaledobject.UtilizationTrigger
)

type Mode struct {
	HTTP *HTTPMode
}

type Kind string

const (
	KindHTTP Kind = "http"
	KindProc Kind = "proc"
)

// ProcessType hints which application-level process runtime a workload
// implements. Empty means unspecified.
type ProcessType string

const (
	ProcessZAMQPConsumer ProcessType = "zamqp-consumer"
	ProcessZAPI          ProcessType = "zapi"
)

type Args struct {
	Env       env.Env
	Namespace string
	Name      string

	Image   string
	Tag     string
	Command []string
	Args    []string
	Profile profile.Profile

	Conf  Conf
	Files Files

	Metrics bool

	// ProcessType hints which application-level process runtime this workload
	// uses (for example a zamqp consumer or zapi HTTP server).
	ProcessType ProcessType

	// Autoscale, when set, emits a KEDA ScaledObject for this workload.
	Autoscale *Autoscale

	// Cluster supplies registered ingress classes for HTTP workloads.
	Cluster *infra.Cluster

	PublicDomains []string // HTTP: <name>.<ns>.<suffix> each
	Veneers       []string // extra public hostnames verbatim

	Mode Mode
}

type Deployment struct {
	pulumi.ResourceState

	PublicHostnames []string
	PrivateHostname string
	ProcessType     ProcessType
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Deployment, error) {
	if args == nil {
		return nil, fmt.Errorf("%s: args is required", typeToken)
	}
	if err := args.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	resourceName := tokens.Qualify(args.Namespace, name)

	comp := &Deployment{}
	if err := ctx.RegisterComponentResource(typeToken, resourceName, comp, opts...); err != nil {
		return nil, fmt.Errorf("registering %s: %w", typeToken, err)
	}

	comp.PublicHostnames = publicHostnames(args.Name, args.Namespace, args.PublicDomains, args.Veneers)
	comp.PrivateHostname = privateHostname(args.Name, args.Namespace)
	comp.ProcessType = args.ProcessType

	kind := selectKind(args.Mode)
	switch kind {
	case KindHTTP:
		if _, err := http.New(ctx, name, &http.Args{
			Env:       args.Env,
			Namespace: args.Namespace,
			Name:      args.Name,
			Image:     args.Image,
			Tag:       args.Tag,
			Command:   args.Command,
			Args:      args.Args,
			Profile:   args.Profile,
			Conf:      args.Conf,
			Files:     args.Files,
			Setup:     *args.Mode.HTTP,
			Metrics:   args.Metrics,
			Internal: http.Internal{
				PublicHostnames: synthesizedPublicHostnames(args.Name, args.Namespace, args.PublicDomains),
				PrivateHostname: comp.PrivateHostname,
				VeneerHostnames: args.Veneers,
			},
			Autoscale: args.Autoscale,
			Cluster:   args.Cluster,
		}, pulumi.Parent(comp)); err != nil {
			return nil, fmt.Errorf("%s: %w", typeToken, err)
		}

	case KindProc:
		if _, err := proc.New(ctx, name, &proc.Args{
			Env:       args.Env,
			Namespace: args.Namespace,
			Name:      args.Name,
			Image:     args.Image,
			Tag:       args.Tag,
			Command:   args.Command,
			Args:      args.Args,
			Profile:   args.Profile,
			Conf:      args.Conf,
			Files:     args.Files,
			Metrics:   args.Metrics,
			Autoscale: args.Autoscale,
			Cluster:   args.Cluster,
		}, pulumi.Parent(comp)); err != nil {
			return nil, fmt.Errorf("%s: %w", typeToken, err)
		}
	}

	if args.ProcessType != "" {
		if err := registerProcessDashboard(ctx, resourceName, args, comp); err != nil {
			return nil, fmt.Errorf("%s: dashboard: %w", typeToken, err)
		}
	}

	publicHostnamesOut := make(pulumi.StringArray, 0, len(comp.PublicHostnames))
	for _, h := range comp.PublicHostnames {
		publicHostnamesOut = append(publicHostnamesOut, pulumi.String(h))
	}
	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"publicHostnames": publicHostnamesOut,
		"privateHostname": pulumi.String(comp.PrivateHostname),
		"processType":     pulumi.String(string(comp.ProcessType)),
	}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

func selectKind(m Mode) Kind {
	if m.HTTP != nil {
		return KindHTTP
	}

	return KindProc
}

func publicHostnames(name, namespace string, domains, veneers []string) []string {
	out := synthesizedPublicHostnames(name, namespace, domains)
	out = append(out, veneers...)

	return out
}

func synthesizedPublicHostnames(name, namespace string, domains []string) []string {
	out := make([]string, 0, len(domains))
	for _, d := range domains {
		out = append(out, name+"."+namespace+"."+d)
	}

	return out
}

func privateHostname(name, namespace string) string {
	return name + "." + namespace + ".svc.cluster.local"
}

func (a *Args) validate() error {
	if a.Name == "" {
		return fmt.Errorf("Name is required")
	}
	if a.Namespace == "" {
		return fmt.Errorf("Namespace is required")
	}
	if a.Image == "" {
		return fmt.Errorf("Image is required")
	}
	if a.Tag == "" {
		return fmt.Errorf("Tag is required")
	}
	if err := a.Env.Validate(); err != nil {
		return err
	}
	if a.Mode.HTTP != nil {
		s := a.Mode.HTTP
		if s.Port == 0 {
			return fmt.Errorf("Mode.HTTP.Port is required")
		}
		if s.Health == "" {
			return fmt.Errorf("Mode.HTTP.Health is required")
		}
	}
	if err := a.ProcessType.validate(); err != nil {
		return err
	}
	if a.ProcessType != "" {
		if !a.Metrics {
			return fmt.Errorf("Metrics must be true when ProcessType is set")
		}
		if a.Cluster == nil || a.Cluster.Grafana == nil {
			return fmt.Errorf("Cluster.Grafana is required when ProcessType is set")
		}
	}

	return nil
}

func registerProcessDashboard(ctx *pulumi.Context, resourceName string, args *Args, parent pulumi.Resource) error {
	err := dashboard.Register(ctx, resourceName, dashboard.Spec{
		Env:       args.Env,
		Namespace: args.Namespace,
		Name:      args.Name,
		Process:   string(args.ProcessType),
	}, args.Cluster.Grafana, parent)
	if err != nil {
		return fmt.Errorf("registering process dashboard: %w", err)
	}

	return nil
}

func (p ProcessType) validate() error {
	switch p {
	case "", ProcessZAMQPConsumer, ProcessZAPI:
		return nil
	default:
		return fmt.Errorf("ProcessType %q is invalid (want zamqp-consumer, zapi, or empty)", p)
	}
}
