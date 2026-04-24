// Package deployment selects between the available workload modes (HTTP
// and background proc) and exposes a uniform surface to its callers.
//
// The polymorphism is the point: callers declare intent via Mode.HTTP
// being set-or-unset, and this package maps that intent to the correct
// underlying ComponentResource. Hostname derivations are centralized
// here so the two modes produce identical hostname outputs whenever a
// caller chooses to read them back.
package deployment

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/k8s/deployment/http"
	"github.com/milagre/zote/pulumi/k8s/deployment/proc"
	"github.com/milagre/zote/pulumi/k8s/internal/podspec"
	"github.com/milagre/zote/pulumi/profile"
	"github.com/milagre/zote/pulumi/tokens"
)

var typeToken = tokens.Token("k8s", "Deployment")

// Conf is re-exported from the pod-spec input layer so callers never
// need to import k8s/internal/*.
type Conf = podspec.Conf

// Files is re-exported for the same reason as Conf.
type Files = podspec.Files

// HTTPMode is the HTTPMode-mode setup shape exposed through this facade.
type HTTPMode = http.Options

// Mode selects which workload kind is materialized. Today only HTTP is
// a first-class choice: setting HTTP picks the HTTP mode and leaving it
// nil picks the background proc mode.
type Mode struct {
	HTTP *HTTPMode
}

// Kind names the runtime-selected workload type. It is the output of the
// pure selectKind function used both at registration time and in tests.
type Kind string

const (
	KindHTTP Kind = "http"
	KindProc Kind = "proc"
)

// Args is the full input set of the facade. Args.Args is the container
// argv; Args.Command is argv[0..]'s entrypoint override, matching Kubernetes
// semantics exactly.
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

	PrometheusMonitored bool

	// PublicDomains is the list of domain suffixes at which the
	// workload should be reachable. The library combines each domain
	// with Name and Namespace to produce one hostname per domain; the
	// public-domain feature is active only in HTTP mode.
	PublicDomains []string
	// Veneers is an additional list of pre-formed hostnames served by
	// the public ingress without generating new certificates for them.
	// Unlike PublicDomains these strings are used verbatim.
	Veneers []string

	Mode Mode
}

// Deployment is the facade component resource. PublicHostnames and
// PrivateHostname are always populated regardless of the chosen mode
// so callers reading the outputs don't have to special-case on it. The
// child workload resource itself is not re-exposed here: callers that
// need to express ordering depend on the facade (a ComponentResource),
// which transitively covers every underlying resource.
type Deployment struct {
	pulumi.ResourceState

	PublicHostnames []string
	PrivateHostname string
}

// New registers the selected underlying workload as a child of the
// facade, so the Pulumi resource graph visibly groups the workload and
// any ingress/service resources under a single parent.
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

	kind := selectKind(args.Mode)
	switch kind {
	case KindHTTP:
		if _, err := http.New(ctx, name, &http.Args{
			Env:                 args.Env,
			Namespace:           args.Namespace,
			Name:                args.Name,
			Image:               args.Image,
			Tag:                 args.Tag,
			Command:             args.Command,
			Args:                args.Args,
			Profile:             args.Profile,
			Conf:                args.Conf,
			Files:               args.Files,
			Setup:               *args.Mode.HTTP,
			PrometheusMonitored: args.PrometheusMonitored,
			Internal: http.Internal{
				PublicHostnames: synthesizedPublicHostnames(args.Name, args.Namespace, args.PublicDomains),
				PrivateHostname: comp.PrivateHostname,
				VeneerHostnames: args.Veneers,
			},
		}, pulumi.Parent(comp)); err != nil {
			return nil, fmt.Errorf("%s: %w", typeToken, err)
		}

	case KindProc:
		if _, err := proc.New(ctx, name, &proc.Args{
			Env:                 args.Env,
			Namespace:           args.Namespace,
			Name:                args.Name,
			Image:               args.Image,
			Tag:                 args.Tag,
			Command:             args.Command,
			Args:                args.Args,
			Profile:             args.Profile,
			Conf:                args.Conf,
			Files:               args.Files,
			PrometheusMonitored: args.PrometheusMonitored,
		}, pulumi.Parent(comp)); err != nil {
			return nil, fmt.Errorf("%s: %w", typeToken, err)
		}
	}

	publicHostnamesOut := make(pulumi.StringArray, 0, len(comp.PublicHostnames))
	for _, h := range comp.PublicHostnames {
		publicHostnamesOut = append(publicHostnamesOut, pulumi.String(h))
	}
	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{
		"publicHostnames": publicHostnamesOut,
		"privateHostname": pulumi.String(comp.PrivateHostname),
	}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

// selectKind maps Mode to the workload that will be constructed. It is
// a pure function so the branching rules can be covered by tests
// without touching Pulumi.
func selectKind(m Mode) Kind {
	if m.HTTP != nil {
		return KindHTTP
	}

	return KindProc
}

// publicHostnames composes the outward-facing hostnames for the
// workload: one hostname per public domain, synthesized from Name and
// Namespace, followed by any verbatim veneer hostnames the caller
// provides. The operation is pure so the output is reproducible from
// its inputs (useful for both testing and offline reasoning).
func publicHostnames(name, namespace string, domains, veneers []string) []string {
	out := synthesizedPublicHostnames(name, namespace, domains)
	out = append(out, veneers...)

	return out
}

// synthesizedPublicHostnames is the subset of publicHostnames that
// derives from PublicDomains alone (veneer strings are used verbatim in
// a separate slot by consumers that need to tell them apart).
func synthesizedPublicHostnames(name, namespace string, domains []string) []string {
	out := make([]string, 0, len(domains))
	for _, d := range domains {
		out = append(out, name+"."+namespace+"."+d)
	}

	return out
}

// privateHostname is the in-cluster DNS name for the workload, matching
// the Kubernetes service DNS convention.
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

	return nil
}
