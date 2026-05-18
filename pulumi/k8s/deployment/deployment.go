// Package deployment is the HTTP vs proc workload facade; hostnames are unified here.
package deployment

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/k8s/deployment/http"
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
)

type Mode struct {
	HTTP *HTTPMode
}

type Kind string

const (
	KindHTTP Kind = "http"
	KindProc Kind = "proc"
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

	PublicDomains []string // HTTP: <name>.<ns>.<suffix> each
	Veneers       []string // extra public hostnames verbatim

	Mode Mode
}

type Deployment struct {
	pulumi.ResourceState

	PublicHostnames []string
	PrivateHostname string
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

	return nil
}
