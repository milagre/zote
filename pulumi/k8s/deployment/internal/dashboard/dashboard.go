// Package dashboard renders and registers Grafana dashboards for workloads
// that declare a [deployment.ProcessType].
package dashboard

import (
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
	default:
		return nil, fmt.Errorf("unsupported process type %q", spec.Process)
	}

	return replacements, nil
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
