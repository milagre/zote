// Package grafana installs upstream Grafana (the dashboard/UI).
package grafana

import (
	"fmt"
	"net/url"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/infra"
	"github.com/milagre/zote/pulumi/internal/helm"
	"github.com/milagre/zote/pulumi/util/annotations"
	"github.com/milagre/zote/pulumi/util/endpoint"
	"github.com/milagre/zote/pulumi/util/stringdata"
	"github.com/milagre/zote/pulumi/util/tokens"
)

const persistenceSize = "1Gi"

var (
	typeToken = tokens.Token("infra", "Grafana")
	helmSpec  = helm.ChartSpec{
		TypeToken:      tokens.Token("infra", "GrafanaHelm"),
		Chart:          "grafana",
		Repository:     "https://grafana-community.github.io/helm-charts",
		DefaultVersion: "12.1.1",
	}
)

type Args struct {
	Env       env.Env
	Namespace string

	Config Config

	// PublicDomains lists public DNS suffixes served by Grafana ingress. When
	// non-empty, [Grafana.API] and [Grafana.UI] target the public hostname
	// (<name>.<namespace>.<domain>) so callers outside the cluster can reach the API.
	PublicDomains []string

	// IngressDeps are resources Ingress creation must wait on (cert-manager, ingress
	// controllers, …). Pass to DependsOn.
	IngressDeps []pulumi.Resource

	// Cluster supplies registered ingress classes and the TLS issuer for autodiscovery.
	Cluster *infra.Cluster

	// Datasources is optional chart `datasources:` values (e.g. built from Loki/Mimir URLs). Not loaded from Config YAML.
	Datasources map[string]any
}

// Grafana is the umbrella component; chart workload and client ConfigMaps/Secrets are children. DependsOn the Helm release via [Grafana.Helm].
type Grafana struct {
	pulumi.ResourceState

	Helm helm.ChartComponent

	K8s K8s

	UI    url.URL
	API   url.URL
	Admin Credentials

	// Ingresses are the public/tunnel Ingress resources fronting Grafana. Callers
	// that reach Grafana through [API] (e.g. the Pulumi Grafana provider) should
	// DependsOn these so work is sequenced after the API becomes routable.
	Ingresses []pulumi.Resource
}

type Credentials struct {
	Username pulumi.StringOutput
	Password pulumi.StringOutput
}

type K8s struct {
	ConfigMap pulumi.StringOutput
	Secret    pulumi.StringOutput
}

func New(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*Grafana, error) {
	if args == nil {
		return nil, fmt.Errorf("grafana: args is required")
	}
	if args.Namespace == "" {
		return nil, fmt.Errorf("grafana: Namespace is required")
	}
	if err := args.Env.Validate(); err != nil {
		return nil, fmt.Errorf("grafana: env: %w", err)
	}
	if err := args.Config.Validate(); err != nil {
		return nil, fmt.Errorf("grafana: config: %w", err)
	}

	comp := &Grafana{}
	rootName := tokens.Qualify(args.Namespace, name)
	if err := ctx.RegisterComponentResource(typeToken, rootName, comp, opts...); err != nil {
		return nil, fmt.Errorf("grafana: register: %w", err)
	}

	helmLogical := tokens.Qualify(args.Namespace, name+"-helm")
	if err := helm.RegisterChartComponentNamed(ctx, helmLogical, helmSpec, &comp.Helm, pulumi.Parent(comp)); err != nil {
		return nil, fmt.Errorf("grafana: helm component: %w", err)
	}

	adminUser := "admin"
	adminPassword, err := random.NewRandomPassword(ctx, name+"-admin", &random.RandomPasswordArgs{
		Length:          pulumi.Int(64),
		Numeric:         pulumi.Bool(true),
		Upper:           pulumi.Bool(true),
		Lower:           pulumi.Bool(true),
		Special:         pulumi.Bool(false),
		MinNumeric:      pulumi.Int(8),
		MinLower:        pulumi.Int(8),
		MinUpper:        pulumi.Int(8),
		OverrideSpecial: pulumi.String("$%&*()-_=+[]{}<>:?"),
		Keepers:         args.Env.RandomKeepers(nil),
	},
		pulumi.Parent(comp),
		pulumi.IgnoreChanges([]string{"*"}),
	)
	if err != nil {
		return nil, fmt.Errorf("grafana: generating admin password: %w", err)
	}

	values := pulumi.Map{
		"adminUser":     pulumi.String(adminUser),
		"adminPassword": adminPassword.Result,
		// Persist Grafana's SQLite DB so dashboards pushed via the API (and org/prefs)
		// survive pod restarts; without a PVC the chart uses an emptyDir that is wiped
		// on every restart. Recreate avoids a ReadWriteOnce multi-attach hang on rollout.
		"persistence": pulumi.Map{
			"enabled": pulumi.Bool(true),
			"size":    pulumi.String(persistenceSize),
		},
		"deploymentStrategy": pulumi.Map{
			"type": pulumi.String("Recreate"),
		},
	}
	if args.Datasources != nil {
		values["datasources"] = helm.Values(args.Datasources)
	}

	if err := helm.InstallChart(ctx, args.Namespace, name, helmSpec, &helm.ChartArgs{
		Namespace: args.Namespace,
		Version:   helm.OptionalChartVersion(args.Config.Version),
		Values:    values,
	}, &comp.Helm); err != nil {
		return nil, fmt.Errorf("grafana: helm install: %w", err)
	}

	cfgPrefix := fmt.Sprintf("%s_GRAFANA", args.Env.Prefix)
	patchForce := pulumi.StringMap{
		annotations.PatchForceKey: pulumi.String("true"),
	}

	// Must not use the Helm release name as the K8s object name: the chart owns
	// Secret/ConfigMap resources named like the release (e.g. admin Secret "grafana").
	clientK8sName := name + "-client"

	svcHost := fmt.Sprintf("%s.%s.svc.cluster.local", name, args.Namespace)
	httpBase := endpoint.HTTP(svcHost, "80", "/")

	cm, err := corev1.NewConfigMap(ctx, name+"-client", &corev1.ConfigMapArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(clientK8sName),
			Namespace:   pulumi.String(args.Namespace),
			Annotations: patchForce,
		},
		Data: pulumi.StringMap{
			cfgPrefix + "_HOST": pulumi.String(httpBase.Hostname()),
			cfgPrefix + "_PORT": pulumi.String(httpBase.Port()),
			cfgPrefix + "_URL":  pulumi.String(httpBase.String()),
		},
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("grafana: client configmap: %w", err)
	}

	sec, err := corev1.NewSecret(ctx, name+"-client", &corev1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:        pulumi.String(clientK8sName),
			Namespace:   pulumi.String(args.Namespace),
			Annotations: patchForce,
		},
		Type: pulumi.String("Opaque"),
		Data: stringdata.SecretData(map[string]pulumi.StringOutput{
			cfgPrefix + "_ADMIN_USER": pulumi.String(adminUser).ToStringOutput(),
			cfgPrefix + "_ADMIN_PASS": adminPassword.Result,
		}),
	}, pulumi.Parent(comp))
	if err != nil {
		return nil, fmt.Errorf("grafana: client secret: %w", err)
	}

	comp.K8s = K8s{
		ConfigMap: cm.Metadata.Name().Elem(),
		Secret:    sec.Metadata.Name().Elem(),
	}
	comp.Admin = Credentials{
		Username: pulumi.String(adminUser).ToStringOutput(),
		Password: adminPassword.Result,
	}

	// The reachable endpoint depends on what Ingress was actually provisioned,
	// so derive UI/API from registerIngresses rather than recomputing the host.
	endpoint, ingresses, err := registerIngresses(ctx, name, args, comp, comp, httpBase)
	if err != nil {
		return nil, fmt.Errorf("grafana: ingress: %w", err)
	}

	comp.UI = endpoint
	comp.API = endpoint
	comp.Ingresses = ingresses

	return comp, nil
}
