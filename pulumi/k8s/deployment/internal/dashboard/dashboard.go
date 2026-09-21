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

	if spec.Capacity > 0 {
		out, err = drawScaleBounds(out, scaleSignalMetric(spec), spec.Capacity, spec.Target)
		if err != nil {
			return "", fmt.Errorf("drawing scale bounds: %w", err)
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
		"__TITLE__":      title,
		"__UID__":        uid,
		"__DATASOURCE__": mimirDatasource,
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

// drawScaleBounds marks target and capacity as dashed threshold lines on every
// panel charting metric, and extends its axis to capacity so the lines stay in
// view while traffic sits far below them.
func drawScaleBounds(dashboardJSON, metric string, capacity, target float64) (string, error) {
	var dashboard map[string]any
	if err := json.Unmarshal([]byte(dashboardJSON), &dashboard); err != nil {
		return "", fmt.Errorf("parsing dashboard: %w", err)
	}

	panels, _ := dashboard["panels"].([]any)
	for _, p := range panels {
		panel, ok := p.(map[string]any)
		if !ok || !panelCharts(panel, metric) {
			continue
		}

		defaults, ok := nestedMap(panel, "fieldConfig", "defaults")
		if !ok {
			return "", fmt.Errorf("panel charting %s has no field defaults", metric)
		}
		custom, ok := nestedMap(defaults, "custom")
		if !ok {
			return "", fmt.Errorf("panel charting %s has no custom field config", metric)
		}

		custom["axisSoftMax"] = capacity
		custom["thresholdsStyle"] = map[string]any{"mode": "dashed"}
		defaults["thresholds"] = map[string]any{
			"mode": "absolute",
			"steps": []any{
				map[string]any{"color": "green", "value": nil},
				map[string]any{"color": "orange", "value": target},
				map[string]any{"color": "red", "value": capacity},
			},
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
