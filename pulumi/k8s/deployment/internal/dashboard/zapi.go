package dashboard

import (
	"github.com/milagre/zote/go/zapi"
	"github.com/milagre/zote/go/zstats"
	"github.com/milagre/zote/go/zstats/zprometheus"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/k8s/internal/podspec"
)

// ZAPIRequestsMetric returns the Prometheus metric name a zapi HTTP server
// publishes for handled requests.
func ZAPIRequestsMetric(e env.Env, namespace, name string) string {
	return zapiMetric(e, namespace, name, "requests")
}

// ZAPIResponsesMetric returns the Prometheus metric name a zapi HTTP server
// publishes for emitted responses.
func ZAPIResponsesMetric(e env.Env, namespace, name string) string {
	return zapiMetric(e, namespace, name, "responses")
}

// ZAPIConcurrencyMetric returns the Prometheus metric name a zapi HTTP server
// publishes for requests in flight.
func ZAPIConcurrencyMetric(e env.Env, namespace, name string) string {
	return zprometheus.MetricName(
		zapi.ConcurrencyStatName(podspec.StatsPrefix(e, namespace, name)),
	)
}

func zapiMetric(e env.Env, namespace, name, metric string) string {
	return zprometheus.MetricName(
		zstats.Qualify(
			zstats.Qualify(podspec.StatsPrefix(e, namespace, name), zapi.StatsPrefix),
			metric,
		),
	)
}
