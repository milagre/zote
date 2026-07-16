// Package dashboard registers the shared RabbitMQ Grafana overview dashboard.
package dashboard

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	grafanapulumi "github.com/pulumiverse/pulumi-grafana/sdk/go/grafana"
	"github.com/pulumiverse/pulumi-grafana/sdk/go/grafana/oss"

	"github.com/milagre/zote/pulumi/infra"
)

const mimirDatasource = "Mimir"

//go:embed rmq.json
var template string

var (
	registerOnce sync.Once
	registerErr  error
)

// RegisterOnce creates or updates the shared RabbitMQ dashboard the first time a
// container RabbitMQ deployment runs while Grafana is available.
func RegisterOnce(ctx *pulumi.Context, cluster *infra.Cluster, parent pulumi.Resource) error {
	if cluster == nil || cluster.Grafana == nil {
		return nil
	}

	registerOnce.Do(func() {
		registerErr = register(ctx, cluster.Grafana, parent)
	})

	if registerErr != nil {
		return fmt.Errorf("registering rabbitmq dashboard: %w", registerErr)
	}

	return nil
}

func register(ctx *pulumi.Context, grafana *grafanapulumi.Provider, parent pulumi.Resource) error {
	configJSON, err := render()
	if err != nil {
		return fmt.Errorf("rendering dashboard: %w", err)
	}

	_, err = oss.NewDashboard(ctx, "rabbitmq-dashboard", &oss.DashboardArgs{
		ConfigJson: pulumi.String(configJSON),
		Overwrite:  pulumi.BoolPtr(true),
	}, pulumi.Parent(parent), pulumi.Provider(grafana))
	if err != nil {
		return fmt.Errorf("creating dashboard: %w", err)
	}

	return nil
}

func render() (string, error) {
	out := strings.ReplaceAll(template, "__DATASOURCE__", mimirDatasource)

	if err := validateJSON(out); err != nil {
		return "", fmt.Errorf("rendered dashboard JSON: %w", err)
	}

	return out, nil
}

func validateJSON(raw string) error {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return err
	}

	return nil
}
