// Package dashboard renders and registers Grafana dashboards for workloads
// that declare a [deployment.ProcessType].
package dashboard

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-grafana/sdk/go/grafana"
	"github.com/pulumiverse/pulumi-grafana/sdk/go/grafana/oss"

	"github.com/milagre/zote/pulumi/env"
)

// mimirDatasource is the uid of the provisioned Mimir datasource (see
// grafana_stack); panels reference the datasource by uid so Grafana binds them.
const mimirDatasource = "mimir"

// podsMetric is the kube-state-metrics series counting a Deployment's
// available replicas.
const podsMetric = "kube_deployment_status_replicas_available"

//go:embed zamqp_consumer.json
var zamqpConsumerTemplate string

//go:embed zapi.json
var zapiTemplate string

// Spec identifies a workload dashboard should target.
type Spec struct {
	Env       env.Env
	Namespace string
	Name      string
	Process   string

	// Capacity and Target are the workload's autoscale bounds in the units of
	// the signal it scales on. Zero draws no bounds.
	Capacity float64
	Target   float64

	// MinPods and MaxPods are an autoscaled workload's replica bounds. Zero
	// MaxPods draws no bounds.
	MinPods int
	MaxPods int
}

// Register creates or updates the Grafana dashboard for spec when grafana is
// configured.
func Register(
	ctx *pulumi.Context,
	resourceName string,
	spec Spec,
	grafana *grafana.Provider,
) error {
	configJSON, err := render(spec)
	if err != nil {
		return fmt.Errorf("rendering dashboard: %w", err)
	}

	_, err = oss.NewDashboard(ctx, resourceName+"-dashboard", &oss.DashboardArgs{
		ConfigJson: pulumi.String(configJSON),
		Overwrite:  pulumi.BoolPtr(true),
	}, pulumi.Provider(grafana))
	if err != nil {
		return fmt.Errorf("creating dashboard: %w", err)
	}

	return nil
}

func render(spec Spec) (string, error) {
	template, err := templateFor(spec.Process)
	if err != nil {
		return "", err
	}

	replacements, err := replacementsFor(spec)
	if err != nil {
		return "", err
	}

	out := template
	for placeholder, value := range replacements {
		out = strings.ReplaceAll(out, placeholder, value)
	}

	if bounds := boundsFor(spec); len(bounds) > 0 {
		out, err = drawBounds(out, bounds)
		if err != nil {
			return "", fmt.Errorf("drawing bounds: %w", err)
		}
	}

	if err := validateJSON(out); err != nil {
		return "", fmt.Errorf("rendered dashboard JSON: %w", err)
	}

	return out, nil
}

func templateFor(process string) (string, error) {
	switch process {
	case "zamqp-consumer":
		return zamqpConsumerTemplate, nil
	case "zapi":
		return zapiTemplate, nil
	default:
		return "", fmt.Errorf("unsupported process type %q", process)
	}
}

func replacementsFor(spec Spec) (map[string]string, error) {
	title := dashboardTitle(spec.Namespace, spec.Name)
	uid := dashboardUID(spec.Namespace, spec.Name)

	replacements := map[string]string{
		"__TITLE__":       title,
		"__UID__":         uid,
		"__DATASOURCE__":  mimirDatasource,
		"__NAMESPACE__":   spec.Namespace,
		"__DEPLOYMENT__":  spec.Name,
		"__PODS_METRIC__": podsMetric,
	}

	switch spec.Process {
	case "zamqp-consumer":
		replacements["__UTILIZATION_METRIC__"] = ZAMQPConsumerUtilizationMetric(spec.Env, spec.Namespace, spec.Name)
		replacements["__RECEIVED_METRIC__"] = ZAMQPConsumerReceivedMetric(spec.Env, spec.Namespace, spec.Name)
	case "zapi":
		replacements["__REQUESTS_METRIC__"] = ZAPIRequestsMetric(spec.Env, spec.Namespace, spec.Name)
		replacements["__RESPONSES_METRIC__"] = ZAPIResponsesMetric(spec.Env, spec.Namespace, spec.Name)
		replacements["__BUSY_SECONDS_METRIC__"] = ZAPIBusySecondsMetric(spec.Env, spec.Namespace, spec.Name)
	default:
		return nil, fmt.Errorf("unsupported process type %q", spec.Process)
	}

	return replacements, nil
}

// scaleSignalMetric returns the metric a process's autoscale bounds are drawn
// against.
func scaleSignalMetric(spec Spec) string {
	switch spec.Process {
	case "zamqp-consumer":
		return ZAMQPConsumerUtilizationMetric(spec.Env, spec.Namespace, spec.Name)
	default:
		return ZAPIBusySecondsMetric(spec.Env, spec.Namespace, spec.Name)
	}
}

// panelBounds are thresholds drawn on every panel charting metric, in style
// (a Grafana thresholdsStyle mode), with base coloring everything below the
// first bound. The axis is extended to softMax so the bounds stay in view while
// the series sits far below them.
type panelBounds struct {
	metric  string
	softMax float64
	style   string
	base    string
	bounds  []bound
}

type bound struct {
	value float64
	color string
}

func boundsFor(spec Spec) []panelBounds {
	var bounds []panelBounds

	if spec.Capacity > 0 {
		bounds = append(bounds, panelBounds{
			metric:  scaleSignalMetric(spec),
			softMax: spec.Capacity,
			style:   "dashed",
			base:    "green",
			bounds:  []bound{{spec.Target, "orange"}, {spec.Capacity, "red"}},
		})
	}

	// The replica range is a band: transparent outside it, green within.
	if spec.MaxPods > 0 {
		bounds = append(bounds, panelBounds{
			metric:  podsMetric,
			softMax: float64(spec.MaxPods),
			style:   "area",
			base:    "transparent",
			bounds:  []bound{{float64(spec.MinPods), "green"}, {float64(spec.MaxPods), "transparent"}},
		})
	}

	return bounds
}

func drawBounds(dashboardJSON string, bounds []panelBounds) (string, error) {
	var dashboard map[string]any
	if err := json.Unmarshal([]byte(dashboardJSON), &dashboard); err != nil {
		return "", fmt.Errorf("parsing dashboard: %w", err)
	}

	panels, _ := dashboard["panels"].([]any)
	for _, p := range panels {
		panel, ok := p.(map[string]any)
		if !ok {
			continue
		}

		for _, b := range bounds {
			if !panelCharts(panel, b.metric) {
				continue
			}

			err := drawPanelBounds(panel, b)
			if err != nil {
				return "", fmt.Errorf("panel charting %s: %w", b.metric, err)
			}
		}
	}

	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(dashboard); err != nil {
		return "", fmt.Errorf("encoding dashboard: %w", err)
	}

	return out.String(), nil
}

func drawPanelBounds(panel map[string]any, b panelBounds) error {
	defaults, ok := nestedMap(panel, "fieldConfig", "defaults")
	if !ok {
		return fmt.Errorf("no field defaults")
	}
	custom, ok := nestedMap(defaults, "custom")
	if !ok {
		return fmt.Errorf("no custom field config")
	}

	steps := []any{map[string]any{"color": b.base, "value": nil}}
	for _, bound := range b.bounds {
		steps = append(steps, map[string]any{"color": bound.color, "value": bound.value})
	}

	custom["axisSoftMax"] = b.softMax
	custom["thresholdsStyle"] = map[string]any{"mode": b.style}
	defaults["thresholds"] = map[string]any{"mode": "absolute", "steps": steps}

	return nil
}

func panelCharts(panel map[string]any, metric string) bool {
	targets, _ := panel["targets"].([]any)
	for _, t := range targets {
		target, ok := t.(map[string]any)
		if !ok {
			continue
		}

		expr, _ := target["expr"].(string)
		if strings.Contains(expr, metric) {
			return true
		}
	}

	return false
}

func nestedMap(m map[string]any, keys ...string) (map[string]any, bool) {
	for _, key := range keys {
		next, ok := m[key].(map[string]any)
		if !ok {
			return nil, false
		}

		m = next
	}

	return m, true
}

func dashboardUID(namespace, name string) string {
	return namespace + "-" + name
}

func dashboardTitle(namespace, name string) string {
	nsTitle := namespace
	if nsTitle != "" {
		nsTitle = strings.ToUpper(nsTitle[:1]) + nsTitle[1:]
	}

	words := strings.Split(name, "-")
	for i, word := range words {
		if word == "" {
			continue
		}

		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}

	return nsTitle + ": " + strings.Join(words, " ")
}

func validateJSON(raw string) error {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return err
	}

	return nil
}
