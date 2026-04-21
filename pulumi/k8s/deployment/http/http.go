// Package http defines an HTTP workload ComponentResource: a Deployment
// running an HTTP process, a ClusterIP Service in front of it, and the
// ingress resources that expose it to the cluster (private nginx) and
// optionally the public internet (public nginx + cloudflare-tunnel in
// local environments).
//
// The component takes hostnames verbatim from its caller; no domain or
// tenant naming is synthesized here, keeping the library independent of
// any downstream product.
package http

import (
	"fmt"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	networkingv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/tokens"
	"github.com/milagre/zote/pulumi/k8s/deployment/internal/body"
	"github.com/milagre/zote/pulumi/k8s/internal/podspec"
	"github.com/milagre/zote/pulumi/profile"
)

var typeToken = tokens.Token("k8s", "HttpDeployment")

// Conf is the set of in-cluster ConfigMap/Secret references and literal
// env values injected into the workload container.
type Conf = podspec.Conf

// Files mounts configmap data onto the container filesystem.
type Files = podspec.Files

// Options describes how the HTTP process is reached and monitored:
// Port is the container port the application listens on, Health is the
// HTTP path used for the liveness probe, and Freq is the probe period in
// seconds (0 uses a sensible default).
type Options struct {
	Port   int
	Health string
	Freq   int
}

// Internal collects the hostnames the HTTP workload is reachable at.
//
// PublicHostnames is the set of externally routed hostnames served via
// the public ingress (with ACME-issued TLS when the environment is not
// local); PrivateHostname is the single in-cluster name the private
// ingress responds to; VeneerHostnames are additional hostnames that
// should be served off the public ingress without generating public
// certificates for them (useful for internal aliases).
type Internal struct {
	PublicHostnames []string
	PrivateHostname string
	VeneerHostnames []string
}

// Args is the full set of inputs for an HTTP workload.
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

	Setup               Options
	PrometheusMonitored bool
	Internal            Internal
}

// Deployment is the component resource. No outputs are exposed: hostnames are
// supplied by the caller (so it already has them), and the Kubernetes
// resources are identified by well-known names (Name, Name+"-nginx-private",
// etc.) that the caller can reference directly if needed.
type Deployment struct {
	pulumi.ResourceState
}

// New registers an HTTP workload and all of its supporting resources.
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

	ports := []podspec.Port{{
		Name:          "http",
		ContainerPort: args.Setup.Port,
		Protocol:      "TCP",
	}}
	if args.PrometheusMonitored {
		ports = append(ports, podspec.Port{
			Name:          "metrics",
			ContainerPort: 9090,
			Protocol:      "TCP",
		})
	}

	if _, err := body.Register(ctx, resourceName, body.Args{
		Env:       args.Env,
		Name:      args.Name,
		Namespace: args.Namespace,
		Type:      "http",
		Image:     args.Image,
		Tag:       args.Tag,
		Command:   args.Command,
		Args:      args.Args,
		Profile:   args.Profile,
		Conf:      args.Conf,
		Files:     args.Files,
		Ports:     ports,
		HTTPLivenessProbe: &podspec.HTTPLivenessProbe{
			Path: args.Setup.Health,
			Port: args.Setup.Port,
			Freq: args.Setup.Freq,
		},
	}, comp); err != nil {
		return nil, fmt.Errorf("%s: %w", typeToken, err)
	}

	svc, err := registerService(ctx, resourceName, args, comp)
	if err != nil {
		return nil, err
	}

	if err := registerPrivateIngress(ctx, resourceName, args, svc, comp); err != nil {
		return nil, err
	}
	if len(args.Internal.PublicHostnames) > 0 {
		if err := registerPublicIngress(ctx, resourceName, args, svc, comp); err != nil {
			return nil, err
		}
	}
	if args.Env.IsLocal() && len(args.Internal.PublicHostnames) > 0 {
		if err := registerTunnelIngress(ctx, resourceName, args, svc, comp); err != nil {
			return nil, err
		}
	}

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
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
	if a.Setup.Port == 0 {
		return fmt.Errorf("Setup.Port is required")
	}
	if a.Setup.Health == "" {
		return fmt.Errorf("Setup.Health is required")
	}
	if a.Internal.PrivateHostname == "" {
		return fmt.Errorf("Internal.PrivateHostname is required")
	}
	if err := a.Env.Validate(); err != nil {
		return err
	}

	return nil
}

// registerService emits the ClusterIP Service that fronts the HTTP
// workload. Port 80 is used as the external contract; the container
// port is treated as an implementation detail and hidden behind this
// fixed service port.
func registerService(ctx *pulumi.Context, name string, args *Args, parent pulumi.Resource) (*corev1.Service, error) {
	svc, err := corev1.NewService(ctx, name, &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(args.Name),
			Namespace: pulumi.String(args.Namespace),
		},
		Spec: &corev1.ServiceSpecArgs{
			Type: pulumi.String("ClusterIP"),
			Selector: pulumi.StringMap{
				"app":    pulumi.String(args.Name),
				"deploy": pulumi.String("http"),
			},
			Ports: corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Name:       pulumi.String("http"),
					Port:       pulumi.Int(80),
					TargetPort: pulumi.Int(args.Setup.Port),
					Protocol:   pulumi.String("TCP"),
				},
			},
		},
	}, pulumi.Parent(parent))
	if err != nil {
		return nil, fmt.Errorf("service: %w", err)
	}

	return svc, nil
}

// registerPrivateIngress publishes the workload at its private hostname
// through the in-cluster nginx ingress controller. The `nginx.class`
// annotation is kept alongside the typed IngressClassName so older nginx
// controllers that only read the annotation still route traffic.
func registerPrivateIngress(ctx *pulumi.Context, name string, args *Args, svc *corev1.Service, parent pulumi.Resource) error {
	_, err := networkingv1.NewIngress(ctx, name+"-nginx-private", &networkingv1.IngressArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(args.Name + "-nginx-private"),
			Namespace: pulumi.String(args.Namespace),
			Annotations: pulumi.StringMap{
				"kubernetes.io/ingress.class": pulumi.String("nginx"),
			},
		},
		Spec: &networkingv1.IngressSpecArgs{
			IngressClassName: pulumi.String("nginx"),
			Rules: networkingv1.IngressRuleArray{
				hostRule(args.Internal.PrivateHostname, svc),
			},
		},
	}, pulumi.Parent(parent))
	if err != nil {
		return fmt.Errorf("private ingress: %w", err)
	}

	return nil
}

// registerPublicIngress serves the workload's public hostnames (plus
// any internal-aliased veneer hostnames) through the public nginx
// ingress. When the environment is not local, a TLS block is added so
// cert-manager issues per-ingress certificates covering exactly those
// hosts; locally, TLS is left off so plain HTTP suffices for developer
// traffic.
func registerPublicIngress(ctx *pulumi.Context, name string, args *Args, svc *corev1.Service, parent pulumi.Resource) error {
	hosts := append(append([]string{}, args.Internal.PublicHostnames...), args.Internal.VeneerHostnames...)

	rules := make(networkingv1.IngressRuleArray, 0, len(hosts))
	for _, h := range hosts {
		rules = append(rules, hostRule(h, svc))
	}

	spec := &networkingv1.IngressSpecArgs{
		IngressClassName: pulumi.String("nginx"),
		Rules:            rules,
	}
	if !args.Env.IsLocal() {
		tlsHosts := make(pulumi.StringArray, 0, len(hosts))
		for _, h := range hosts {
			tlsHosts = append(tlsHosts, pulumi.String(h))
		}
		spec.Tls = networkingv1.IngressTLSArray{
			&networkingv1.IngressTLSArgs{
				Hosts:      tlsHosts,
				SecretName: pulumi.String(args.Name + "-tls"),
			},
		}
	}

	_, err := networkingv1.NewIngress(ctx, name+"-nginx-public", &networkingv1.IngressArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(args.Name + "-nginx-public"),
			Namespace: pulumi.String(args.Namespace),
			Annotations: pulumi.StringMap{
				"kubernetes.io/ingress.class":    pulumi.String("nginx"),
				"cert-manager.io/cluster-issuer": pulumi.String("letsencrypt-http01"),
			},
		},
		Spec: spec,
	}, pulumi.Parent(parent))
	if err != nil {
		return fmt.Errorf("public ingress: %w", err)
	}

	return nil
}

// registerTunnelIngress exposes the workload through the
// cloudflare-tunnel ingress class so a developer-local cluster can
// respond on real public hostnames without a cloud load balancer.
// Only the public hostnames (not veneers) are routed this way.
func registerTunnelIngress(ctx *pulumi.Context, name string, args *Args, svc *corev1.Service, parent pulumi.Resource) error {
	rules := make(networkingv1.IngressRuleArray, 0, len(args.Internal.PublicHostnames))
	for _, h := range args.Internal.PublicHostnames {
		rules = append(rules, hostRule(h, svc))
	}

	_, err := networkingv1.NewIngress(ctx, name+"-tunnel", &networkingv1.IngressArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(args.Name + "-tunnel"),
			Namespace: pulumi.String(args.Namespace),
			Annotations: pulumi.StringMap{
				"kubernetes.io/ingress.class": pulumi.String("cloudflare-tunnel"),
			},
		},
		Spec: &networkingv1.IngressSpecArgs{
			IngressClassName: pulumi.String("cloudflare-tunnel"),
			Rules:            rules,
		},
	}, pulumi.Parent(parent))
	if err != nil {
		return fmt.Errorf("tunnel ingress: %w", err)
	}

	return nil
}

// hostRule is the common Ingress rule shape: a single host, a prefix
// path of "/", and a backend pointing at the workload's Service on port
// 80 (matching the port exposed by registerService).
func hostRule(host string, svc *corev1.Service) *networkingv1.IngressRuleArgs {
	pathType := "Prefix"
	return &networkingv1.IngressRuleArgs{
		Host: pulumi.String(host),
		Http: &networkingv1.HTTPIngressRuleValueArgs{
			Paths: networkingv1.HTTPIngressPathArray{
				&networkingv1.HTTPIngressPathArgs{
					Path:     pulumi.String("/"),
					PathType: pulumi.String(pathType),
					Backend: &networkingv1.IngressBackendArgs{
						Service: &networkingv1.IngressServiceBackendArgs{
							Name: svc.Metadata.Name().Elem(),
							Port: &networkingv1.ServiceBackendPortArgs{
								Number: pulumi.Int(80),
							},
						},
					},
				},
			},
		},
	}
}
