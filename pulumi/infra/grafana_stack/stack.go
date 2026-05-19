// Package grafana_stack installs the full Grafana observability stack:
// dashboard (Grafana), metrics (Mimir), logs (Loki), and agents (Alloy).
//
// The subpackages under infra/grafana_stack/* remain independently installable.
package grafana_stack

import (
	"fmt"
	"net/url"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	grafanapulumi "github.com/pulumiverse/pulumi-grafana/sdk/go/grafana"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/infra/grafana_stack/alloy"
	"github.com/milagre/zote/pulumi/infra/grafana_stack/grafana"
	"github.com/milagre/zote/pulumi/infra/grafana_stack/loki"
	"github.com/milagre/zote/pulumi/infra/grafana_stack/mimir"
	"github.com/milagre/zote/pulumi/svc/objectstorage"
	"github.com/milagre/zote/pulumi/util/tokens"
)

var typeToken = tokens.Token("infra", "GrafanaStack")

// Args configures the Grafana stack. It is intentionally high-level:
// subpackages keep their own config and may be used standalone.
type Args struct {
	Env       env.Env
	Namespace string

	Config        Config
	ObjectStorage objectstorage.ObjectStorage

	// Cluster registers deployed capabilities when non-nil.
	Cluster *infra.Cluster
}

type Credentials struct {
	Username pulumi.StringOutput
	Password pulumi.StringOutput
}

type K8s struct {
	ConfigMap pulumi.StringOutput
	Secret    pulumi.StringOutput
}

type Dashboard struct {
	K8s   K8s
	UI    url.URL
	API   url.URL
	Admin Credentials
}

type GrafanaStack struct {
	pulumi.ResourceState

	Dashboard Dashboard

	Grafana *grafana.Grafana
	Loki    *loki.Loki
	Mimir   *mimir.Mimir
	Alloy   *alloy.Alloy
}

// New is the single entry point for deploying the entire Grafana stack.
func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*GrafanaStack, error) {
	if args == nil {
		return nil, fmt.Errorf("%s: args is required", typeToken)
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("%s: Namespace is required", typeToken)
	}
	if err := args.Env.Validate(); err != nil {
		return nil, fmt.Errorf("%s: env: %w", typeToken, err)
	}
	if err := args.Config.Validate(); err != nil {
		return nil, fmt.Errorf("%s: config: %w", typeToken, err)
	}
	resourceName := tokens.Qualify(args.Namespace, name)
	comp := &GrafanaStack{}
	if err := ctx.RegisterComponentResource(typeToken, resourceName, comp, opts...); err != nil {
		return nil, fmt.Errorf("%s: register: %w", typeToken, err)
	}

	lk, err := loki.New(ctx, "loki", &loki.Args{
		Env:           args.Env,
		Namespace:     args.Namespace,
		Config:        *args.Config.Loki,
		ObjectStorage: args.ObjectStorage,
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: loki: %w", typeToken, err)
	}

	mm, err := mimir.New(ctx, "mimir", &mimir.Args{
		Env:           args.Env,
		Namespace:     args.Namespace,
		Config:        *args.Config.Mimir,
		ObjectStorage: args.ObjectStorage,
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: mimir: %w", typeToken, err)
	}

	al, err := alloy.New(ctx, "alloy", &alloy.Args{
		Namespace: args.Namespace,
		Config:    *args.Config.Alloy,
		River:     defaultAlloyRiver(lk.PushURL, mm.PushURL),
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: alloy: %w", typeToken, err)
	}

	g, err := grafana.New(ctx, "grafana", &grafana.Args{
		Env:         args.Env,
		Namespace:   args.Namespace,
		Config:      *args.Config.Dashboard,
		Datasources: defaultGrafanaDatasources(lk.Gateway.String(), mm.Prometheus.String()),
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("%s: grafana: %w", typeToken, err)
	}

	comp.Grafana = g
	comp.Loki = lk
	comp.Mimir = mm
	comp.Alloy = al

	comp.Dashboard = Dashboard{
		K8s: K8s{
			ConfigMap: g.K8s.ConfigMap,
			Secret:    g.K8s.Secret,
		},
		UI:  g.UI,
		API: g.API,
		Admin: Credentials{
			Username: g.Admin.Username,
			Password: g.Admin.Password,
		},
	}

	if err := ctx.RegisterResourceOutputs(comp, pulumi.Map{}); err != nil {
		return nil, fmt.Errorf("%s: registering outputs: %w", typeToken, err)
	}

	prov, err := newGrafanaProvider(ctx, resourceName, g, comp)
	if err != nil {
		return nil, fmt.Errorf("%s: grafana provider: %w", typeToken, err)
	}

	args.Cluster.SetGrafana(prov)

	return comp, nil
}

// newGrafanaProvider configures the Pulumiverse Grafana provider against the
// in-cluster dashboard API (basic auth admin credentials from the Helm release).
func newGrafanaProvider(ctx *pulumi.Context, name string, g *grafana.Grafana, parent pulumi.Resource) (*grafanapulumi.Provider, error) {
	auth := pulumi.All(g.Admin.Username, g.Admin.Password).ApplyT(func(vals []any) (string, error) {
		return fmt.Sprintf("%s:%s", vals[0].(string), vals[1].(string)), nil
	}).(pulumi.StringOutput)

	return grafanapulumi.NewProvider(ctx, name+"-provider", &grafanapulumi.ProviderArgs{
		Url:  pulumi.String(g.API.String()),
		Auth: auth,
	}, pulumi.Parent(parent), pulumi.DependsOn([]pulumi.Resource{g.Helm.Release}))
}

func defaultGrafanaDatasources(lokiBaseURL string, mimirPrometheusURL string) map[string]any {
	return map[string]any{
		"datasources.yaml": map[string]any{
			"apiVersion": 1,
			"datasources": []any{
				map[string]any{
					"name":   "Loki",
					"type":   "loki",
					"access": "proxy",
					"url":    lokiBaseURL,
				},
				map[string]any{
					"name":   "Mimir",
					"type":   "prometheus",
					"access": "proxy",
					"url":    mimirPrometheusURL,
				},
			},
		},
	}
}

// defaultAlloyRiver builds Alloy River config from [loki.Loki.PushURL] and [mimir.Mimir.PushURL].
func defaultAlloyRiver(lokiPush, mimirPush pulumi.StringOutput) pulumi.StringOutput {
	return pulumi.All(lokiPush, mimirPush).ApplyT(func(vals []interface{}) string {
		lk := vals[0].(string)
		mm := vals[1].(string)

		return fmt.Sprintf(defaultAlloyRiverTemplate, mm, lk)
	}).(pulumi.StringOutput)
}

const defaultAlloyRiverTemplate = `logging {
  level  = "info"
  format = "logfmt"
}

prometheus.remote_write "default" {
  endpoint {
    url = "%s"
  }
}

// Prometheus Operator CRDs: PodMonitors + ServiceMonitors.
prometheus.operator.podmonitors "default" {
  forward_to = [prometheus.remote_write.default.receiver]
}

prometheus.operator.servicemonitors "default" {
  forward_to = [prometheus.remote_write.default.receiver]
}

// Annotation-based scraping (covers ReplicaSets via pod template annotations):
// - prometheus.io/scrape: "true"
// - prometheus.io/path (optional)
// - prometheus.io/port (optional)
discovery.kubernetes "pods" {
  role = "pod"
}

discovery.relabel "pods_scrape" {
  targets = discovery.kubernetes.pods.targets

  rule {
    source_labels = ["__meta_kubernetes_pod_annotation_prometheus_io_scrape"]
    action        = "keep"
    regex         = "true"
  }

  rule {
    source_labels = ["__meta_kubernetes_pod_annotation_prometheus_io_path"]
    action        = "replace"
    target_label  = "__metrics_path__"
    regex         = "(.+)"
  }

  rule {
    source_labels = ["__address__", "__meta_kubernetes_pod_annotation_prometheus_io_port"]
    action        = "replace"
    target_label  = "__address__"
    regex         = "([^:]+)(?::\\d+)?;(\\d+)"
    replacement   = "$1:$2"
  }
}

prometheus.scrape "pods_annotations" {
  targets    = discovery.relabel.pods_scrape.output
  forward_to = [prometheus.remote_write.default.receiver]
}

// Pod logs (per-node): Grafana Alloy Helm sets HOSTNAME to the node name for field selectors.
discovery.kubernetes "pod_logs" {
  role = "pod"

  selectors {
    role  = "pod"
    field = "spec.nodeName=" + coalesce(sys.env("HOSTNAME"), constants.hostname)
  }
}

discovery.relabel "pod_logs" {
  targets = discovery.kubernetes.pod_logs.targets

  rule {
    source_labels = ["__meta_kubernetes_namespace"]
    action        = "replace"
    target_label  = "namespace"
  }

  rule {
    source_labels = ["__meta_kubernetes_pod_name"]
    action        = "replace"
    target_label  = "pod"
  }

  rule {
    source_labels = ["__meta_kubernetes_pod_container_name"]
    action        = "replace"
    target_label  = "container"
  }

  rule {
    source_labels = ["__meta_kubernetes_pod_node_name"]
    action        = "replace"
    target_label  = "node"
  }

  rule {
    source_labels = ["__meta_kubernetes_pod_label_app_kubernetes_io_name"]
    action        = "replace"
    target_label  = "app"
  }

  rule {
    source_labels = ["__meta_kubernetes_namespace", "__meta_kubernetes_pod_name", "__meta_kubernetes_pod_container_name"]
    separator     = "/"
    regex         = "(.+)"
    replacement   = "$1"
    target_label  = "job"
    action        = "replace"
  }
}

loki.source.kubernetes "pod_logs" {
  targets    = discovery.relabel.pod_logs.output
  forward_to = [loki.write.default.receiver]
}

loki.write "default" {
  endpoint {
    url = "%s"
  }
}
`
