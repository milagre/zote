package dashboard

import (
	"github.com/milagre/zote/go/zamqp"
	"github.com/milagre/zote/go/zstats"
	"github.com/milagre/zote/go/zstats/zprometheus"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/k8s/internal/podspec"
)

// ZAMQPConsumerUtilizationMetric returns the Prometheus metric name a zamqp
// consumer workload publishes for utilization.
func ZAMQPConsumerUtilizationMetric(e env.Env, namespace, name string) string {
	return zprometheus.MetricName(
		zamqp.ConsumerUtilizationStatName(podspec.StatsPrefix(e, namespace, name)),
	)
}

// ZAMQPConsumerReceivedMetric returns the Prometheus metric name a zamqp
// consumer workload publishes for received messages.
func ZAMQPConsumerReceivedMetric(e env.Env, namespace, name string) string {
	return zprometheus.MetricName(
		zstats.Qualify(
			zstats.Qualify(podspec.StatsPrefix(e, namespace, name), zamqp.ConsumerStatsPrefix),
			"received",
		),
	)
}
