// Package http is an HTTP Deployment, Service, and nginx ingress (private + optional public); hostnames from Args.Internal.
package http

import (
	"fmt"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	networkingv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/k8s/deployment/internal/body"
	"github.com/milagre/zote/pulumi/k8s/internal/podspec"
	"github.com/milagre/zote/pulumi/util/annotations"
	"github.com/milagre/zote/pulumi/util/profile"
	"github.com/milagre/zote/pulumi/util/tokens"
)

var typeToken = tokens.Token("k8s", "HttpDeployment")

type (
	Conf  = podspec.Conf
	Files = podspec.Files
)

type Options struct {
	Port   int
	Health string
	Freq   int
}

type Internal struct {
	PublicHostnames []string
	PrivateHostname string
	VeneerHostnames []string
}

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

	Setup    Options
	Metrics  bool
	Internal Internal

	// Cluster supplies registered ingress classes and the TLS issuer for autodiscovery.
	Cluster *infra.Cluster
}

type Deployment struct {
	pulumi.ResourceState
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

	ports := []podspec.Port{{
		Name:          "http",
		ContainerPort: args.Setup.Port,
		Protocol:      "TCP",
	}}
	if args.Metrics {
		ports = append(ports, podspec.Port{
			Name:          "metrics",
			ContainerPort: body.MetricsListenPort,
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
		Metrics:   args.Metrics,
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

	if class := privateIngressClass(args.Cluster); class != nil {
		if err := registerPrivateIngress(ctx, resourceName, args, svc, comp, class); err != nil {
			return nil, err
		}
	}

	hasPublicHosts := len(args.Internal.PublicHostnames) > 0 || len(args.Internal.VeneerHostnames) > 0
	if hasPublicHosts {
		if class := pulumi.StringPtrFromPtr(publicIngressClassName(args.Cluster)); class != nil {
			if err := registerPublicIngress(ctx, resourceName, args, svc, comp, class); err != nil {
				return nil, err
			}
		}
	}

	if args.Env.IsLocal() && len(args.Internal.PublicHostnames) > 0 {
		if class := pulumi.StringPtrFromPtr(tunnelIngressClassName(args.Cluster)); class != nil {
			if err := registerTunnelIngress(ctx, resourceName, args, svc, comp, class); err != nil {
				return nil, err
			}
		}
	}

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	return comp, nil
}

func publicIngressClassName(cluster *infra.Cluster) *string {
	if cluster == nil {
		return nil
	}

	return cluster.PublicIngressClassName
}

func tunnelIngressClassName(cluster *infra.Cluster) *string {
	if cluster == nil {
		return nil
	}

	return cluster.TunnelIngressClassName
}

func clusterIssuerName(cluster *infra.Cluster) *string {
	if cluster == nil {
		return nil
	}

	return cluster.ClusterIssuerName
}

func privateIngressClassName(cluster *infra.Cluster) *string {
	if cluster == nil {
		return nil
	}

	if cluster.PrivateIngressClassName != nil {
		return cluster.PrivateIngressClassName
	}

	return cluster.PublicIngressClassName
}

// privateIngressClass prefers PrivateIngressClassName; until a private controller
// exists, nginx private hostnames use the public class when that is registered.
func privateIngressClass(cluster *infra.Cluster) pulumi.StringPtrInput {
	return pulumi.StringPtrFromPtr(privateIngressClassName(cluster))
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
			Annotations: pulumi.StringMap{
				annotations.SkipAwaitKey: pulumi.String(annotations.SkipAwaitValueAll),
			},
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
func registerPrivateIngress(
	ctx *pulumi.Context,
	name string,
	args *Args,
	svc *corev1.Service,
	parent pulumi.Resource,
	ingressClass pulumi.StringPtrInput,
) error {
	className := privateIngressClassName(args.Cluster)

	_, err := networkingv1.NewIngress(ctx, name+"-nginx-private", &networkingv1.IngressArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(args.Name + "-nginx-private"),
			Namespace: pulumi.String(args.Namespace),
			Annotations: pulumi.StringMap{
				annotations.SkipAwaitKey:      pulumi.String(annotations.SkipAwaitValueAll),
				"kubernetes.io/ingress.class": pulumi.String(*className),
			},
		},
		Spec: &networkingv1.IngressSpecArgs{
			IngressClassName: ingressClass,
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
func registerPublicIngress(
	ctx *pulumi.Context,
	name string,
	args *Args,
	svc *corev1.Service,
	parent pulumi.Resource,
	ingressClass pulumi.StringPtrInput,
) error {
	hosts := append(append([]string{}, args.Internal.PublicHostnames...), args.Internal.VeneerHostnames...)

	rules := make(networkingv1.IngressRuleArray, 0, len(hosts))
	for _, h := range hosts {
		rules = append(rules, hostRule(h, svc))
	}

	className := publicIngressClassName(args.Cluster)
	ann := pulumi.StringMap{
		annotations.SkipAwaitKey:      pulumi.String(annotations.SkipAwaitValueAll),
		"kubernetes.io/ingress.class": pulumi.String(*className),
	}

	spec := &networkingv1.IngressSpecArgs{
		IngressClassName: ingressClass,
		Rules:            rules,
	}
	if !args.Env.IsLocal() {
		issuer := clusterIssuerName(args.Cluster)
		if issuer == nil {
			return nil
		}

		ann["cert-manager.io/cluster-issuer"] = pulumi.String(*issuer)

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
			Name:        pulumi.String(args.Name + "-nginx-public"),
			Namespace:   pulumi.String(args.Namespace),
			Annotations: ann,
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
func registerTunnelIngress(
	ctx *pulumi.Context,
	name string,
	args *Args,
	svc *corev1.Service,
	parent pulumi.Resource,
	ingressClass pulumi.StringPtrInput,
) error {
	className := tunnelIngressClassName(args.Cluster)

	rules := make(networkingv1.IngressRuleArray, 0, len(args.Internal.PublicHostnames))
	for _, h := range args.Internal.PublicHostnames {
		rules = append(rules, hostRule(h, svc))
	}

	_, err := networkingv1.NewIngress(ctx, name+"-tunnel", &networkingv1.IngressArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(args.Name + "-tunnel"),
			Namespace: pulumi.String(args.Namespace),
			Annotations: pulumi.StringMap{
				annotations.SkipAwaitKey:      pulumi.String(annotations.SkipAwaitValueAll),
				"kubernetes.io/ingress.class": pulumi.String(*className),
			},
		},
		Spec: &networkingv1.IngressSpecArgs{
			IngressClassName: ingressClass,
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
