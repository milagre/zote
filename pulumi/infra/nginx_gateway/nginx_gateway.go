// Package nginx_gateway installs NGINX Gateway Fabric, the Gateway API CRDs it
// implements, and the shared Gateway routes attach to. Local environments
// expose the data plane on a NodePort and serve plain HTTP; elsewhere it sits
// behind a LoadBalancer and terminates TLS for each hostname with a
// certificate from the cluster's ClusterIssuer. Sizing via [profile.Profile]
// only.
//
// Helm installs the chart's own CRDs on first install only: a chart upgrade
// that changes them needs them applied by hand.
package nginx_gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	yamlv2 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/yaml/v2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/util/annotations"
	"github.com/milagre/zote/pulumi/util/profile"
	"github.com/milagre/zote/pulumi/util/tokens"
)

const (
	DefaultGatewayClass = "nginx"
	DefaultGatewayName  = "gateway"

	// The Gateway API release the chart's controller implements.
	crdURL    = "https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.7.0/standard-install.yaml"
	crdDigest = "4e2602c4cbe1afbc394d86f4ac6522e1a5263b01133bba1e40a6d1a549e753a1"

	// dataPlaneMetricsPort is where each data plane pod's agent serves
	// Prometheus metrics.
	dataPlaneMetricsPort = "9113"

	// maxListeners is the Gateway API's limit for one Gateway.
	maxListeners = 64
)

var spec = helm.ChartSpec{
	TypeToken:      tokens.Token("infra", "NginxGateway"),
	Chart:          "oci://ghcr.io/nginx/charts/nginx-gateway-fabric",
	DefaultVersion: "2.9.0",
}

type Args struct {
	Namespace string
	Env       env.Env

	// GatewayName names the shared Gateway; empty uses [DefaultGatewayName].
	GatewayName string

	Config Config

	// Cluster receives the Gateway, and supplies the ClusterIssuer when the
	// Gateway is created.
	Cluster *infra.Cluster
}

type NginxGateway struct {
	helm.ChartComponent

	// CRDs installs the Gateway API CRDs; nil with Config.ExternalCRDs.
	CRDs pulumi.Resource

	env     env.Env
	cluster *infra.Cluster
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*NginxGateway, error) {
	if args == nil {
		return nil, fmt.Errorf("nginx_gateway: args is required")
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("nginx_gateway: Namespace is required")
	}
	if args.Cluster == nil {
		return nil, fmt.Errorf("nginx_gateway: Cluster is required")
	}
	prof, err := args.Config.ResourceProfile()
	if err != nil {
		return nil, fmt.Errorf("nginx_gateway: config: %w", err)
	}

	gatewayName := strings.TrimSpace(args.GatewayName)
	if gatewayName == "" {
		gatewayName = DefaultGatewayName
	}

	comp := &NginxGateway{env: args.Env, cluster: args.Cluster}
	if err := helm.RegisterChartComponent(ctx, args.Namespace, name, spec, &comp.ChartComponent, opts...); err != nil {
		return nil, err
	}

	var ready []pulumi.Resource
	if !args.Config.ExternalCRDs {
		crds, err := registerCRDs(ctx, tokens.Qualify(args.Namespace, name)+"-gateway-api-crds", comp)
		if err != nil {
			return nil, err
		}

		comp.CRDs = crds
		ready = append(ready, crds)
	}

	if err := helm.InstallChart(ctx, args.Namespace, name, spec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   helm.OptionalChartVersion(args.Config.Version),
		Values:    values(args, prof, gatewayName),
		DependsOn: ready,
	}, &comp.ChartComponent); err != nil {
		return nil, err
	}

	args.Cluster.SetGateway(gatewayName, args.Namespace, ready...)

	return comp, nil
}

// CreateGateway creates the shared Gateway with a listener for every hostname
// routes have registered, so it must run after the last route is created.
// Where TLS is terminated, plain HTTP only redirects to HTTPS, and it accepts
// routes from the Gateway's own namespace, where ACME HTTP-01 solvers put
// theirs.
func (g *NginxGateway) CreateGateway(ctx *pulumi.Context) error {
	gw := g.cluster.Gateway
	tls := !g.env.IsLocal() && g.cluster.ClusterIssuerName != nil

	httpFrom := "All"
	if tls {
		httpFrom = "Same"
	}

	listeners := pulumi.Array{
		pulumi.Map{
			"name":          pulumi.String(infra.GatewayHTTPListener),
			"port":          pulumi.Int(80),
			"protocol":      pulumi.String("HTTP"),
			"allowedRoutes": allowedRoutes(httpFrom),
		},
	}

	ann := pulumi.StringMap{
		annotations.WaitForKey: pulumi.String(annotations.WaitForValueImmediate),
	}

	if tls {
		hosts := gw.Hostnames()
		if len(hosts)+1 > maxListeners {
			return fmt.Errorf("nginx_gateway: %d hostnames need %d listeners, more than the %d a Gateway holds", len(hosts), len(hosts)+1, maxListeners)
		}

		for _, h := range hosts {
			listeners = append(listeners, pulumi.Map{
				"name":     pulumi.String(infra.GatewayListenerName(h)),
				"hostname": pulumi.String(h),
				"port":     pulumi.Int(443),
				"protocol": pulumi.String("HTTPS"),
				"tls": pulumi.Map{
					"mode": pulumi.String("Terminate"),
					"certificateRefs": pulumi.Array{
						pulumi.Map{
							"kind": pulumi.String("Secret"),
							"name": pulumi.String(h + "-tls"),
						},
					},
				},
				"allowedRoutes": allowedRoutes("All"),
			})
		}

		ann["cert-manager.io/cluster-issuer"] = pulumi.String(*g.cluster.ClusterIssuerName)
	}

	deps := append([]pulumi.Resource{g.Release}, gw.Ready...)

	gateway, err := apiextensions.NewCustomResource(ctx, tokens.Qualify(gw.Namespace, gw.Name), &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("gateway.networking.k8s.io/v1"),
		Kind:       pulumi.String("Gateway"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(gw.Name),
			Namespace:   pulumi.String(gw.Namespace),
			Annotations: ann,
		},
		OtherFields: map[string]any{
			"spec": pulumi.Map{
				"gatewayClassName": pulumi.String(DefaultGatewayClass),
				"listeners":        listeners,
			},
		},
	}, pulumi.Parent(g), pulumi.DependsOn(deps))
	if err != nil {
		return fmt.Errorf("nginx_gateway: gateway: %w", err)
	}

	if !tls {
		return nil
	}

	err = registerHTTPSRedirect(ctx, gw, g, gateway)
	if err != nil {
		return err
	}

	return nil
}

func allowedRoutes(from string) pulumi.Map {
	return pulumi.Map{
		"namespaces": pulumi.Map{"from": pulumi.String(from)},
	}
}

// registerHTTPSRedirect sends every plain HTTP request to HTTPS. An ACME
// solver's route names its hostname and an exact path, so it takes precedence.
func registerHTTPSRedirect(ctx *pulumi.Context, gw *infra.Gateway, parent, gateway pulumi.Resource) error {
	name := gw.Name + "-https-redirect"

	_, err := apiextensions.NewCustomResource(ctx, tokens.Qualify(gw.Namespace, name), &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("gateway.networking.k8s.io/v1"),
		Kind:       pulumi.String("HTTPRoute"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String(name),
			Namespace: pulumi.String(gw.Namespace),
			Annotations: pulumi.StringMap{
				annotations.WaitForKey: pulumi.String(annotations.WaitForValueImmediate),
			},
		},
		OtherFields: map[string]any{
			"spec": pulumi.Map{
				"parentRefs": pulumi.Array{
					pulumi.Map{
						"name":        pulumi.String(gw.Name),
						"sectionName": pulumi.String(infra.GatewayHTTPListener),
					},
				},
				"rules": pulumi.Array{
					pulumi.Map{
						"filters": pulumi.Array{
							pulumi.Map{
								"type": pulumi.String("RequestRedirect"),
								"requestRedirect": pulumi.Map{
									"scheme":     pulumi.String("https"),
									"statusCode": pulumi.Int(301),
								},
							},
						},
					},
				},
			},
		},
	}, pulumi.Parent(parent), pulumi.DependsOn([]pulumi.Resource{gateway}))
	if err != nil {
		return fmt.Errorf("nginx_gateway: https redirect: %w", err)
	}

	return nil
}

// registerCRDs installs the Gateway API CRDs. They are kept when this
// component goes: deleting a CRD deletes every object of its kind in the
// cluster, routes in every namespace included.
func registerCRDs(ctx *pulumi.Context, name string, parent pulumi.Resource) (pulumi.Resource, error) {
	file, err := helm.LocalFile(ctx, crdURL, crdDigest)
	if err != nil {
		return nil, fmt.Errorf("nginx_gateway: gateway api crds: %w", err)
	}

	crds, err := yamlv2.NewConfigGroup(ctx, name, &yamlv2.ConfigGroupArgs{
		Files: pulumi.ToStringArray([]string{file}),
	}, pulumi.Parent(parent), pulumi.RetainOnDelete(true), pulumi.Transforms([]pulumi.ResourceTransform{retainOnDelete}))
	if err != nil {
		return nil, fmt.Errorf("nginx_gateway: gateway api crds: %w", err)
	}

	return crds, nil
}

func retainOnDelete(_ context.Context, a *pulumi.ResourceTransformArgs) *pulumi.ResourceTransformResult {
	a.Opts.RetainOnDelete = true

	return &pulumi.ResourceTransformResult{Props: a.Props, Opts: a.Opts}
}

func values(args *Args, p profile.Profile, gatewayName string) pulumi.Map {
	autoscaling := map[string]any{
		"enable":                            true,
		"minReplicas":                       3,
		"maxReplicas":                       6,
		"targetCPUUtilizationPercentage":    75,
		"targetMemoryUtilizationPercentage": 75,
	}
	if p.Num != nil {
		autoscaling["minReplicas"] = p.Num.Min
		autoscaling["maxReplicas"] = p.Num.Max
	}

	serviceType := "LoadBalancer"
	if args.Env.IsLocal() {
		serviceType = "NodePort"
	}

	nginx := map[string]any{
		"autoscaling": autoscaling,
		// The chart only offers a ServiceMonitor for the data plane; scrape
		// annotations let annotation-based discovery find its pods.
		"patches": []any{
			map[string]any{
				"type": "StrategicMerge",
				"value": map[string]any{
					"spec": map[string]any{
						"template": map[string]any{
							"metadata": map[string]any{
								"annotations": map[string]string{
									"prometheus.io/scrape": "true",
									"prometheus.io/port":   dataPlaneMetricsPort,
								},
							},
						},
					},
				},
			},
		},
		"container": map[string]any{
			"resources": map[string]any{
				"requests": map[string]any{
					"cpu":    p.MinCoresMilli(),
					"memory": p.MinMemMiB(),
				},
				"limits": map[string]any{
					"cpu":    p.MaxCoresMilli(),
					"memory": p.MaxMemMiB(),
				},
			},
		},
		"pod": map[string]any{
			"affinity": map[string]any{
				"podAntiAffinity": map[string]any{
					"preferredDuringSchedulingIgnoredDuringExecution": []any{
						map[string]any{
							"weight": 100,
							"podAffinityTerm": map[string]any{
								"labelSelector": map[string]any{
									"matchLabels": map[string]string{
										"gateway.networking.k8s.io/gateway-name": gatewayName,
									},
								},
								"topologyKey": "kubernetes.io/hostname",
							},
						},
					},
				},
			},
		},
		// Cluster, not the chart's Local: the load balancer then reaches the
		// data plane through any node, whichever nodes its pods run on.
		"service": map[string]any{
			"type":                  serviceType,
			"externalTrafficPolicy": "Cluster",
		},
	}

	return helm.Values(map[string]any{
		"nginxGateway": map[string]any{
			"gatewayClassName": DefaultGatewayClass,
			"productTelemetry": map[string]any{"enable": false},
		},
		"nginx": nginx,
	})
}
