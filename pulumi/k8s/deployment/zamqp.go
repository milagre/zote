package deployment

import (
	"fmt"

	"github.com/milagre/zote/go/zamqp"
	"github.com/milagre/zote/go/zstats/zprometheus"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/k8s/internal/podspec"
)

// ZAMQPUtilizationStat returns the PromQL that averages the zamqp consumer
// utilization gauge a workload publishes, for use as
// [UtilizationTrigger.Query]. Only proc workloads that actually run a zamqp
// consumer expose this metric; a workload autoscaling on any other signal must
// supply its own query.
//
// e, namespace, and name identify the workload exactly as passed to New, so the
// query targets the metric name that workload emits. Both halves of the stored
// name come from the zote runtime rather than being restated here:
//
//   - zamqp.ConsumerUtilizationStatName qualifies the ambient stats prefix with
//     the consumer prefix and metric the direct consumer emits, and
//   - zprometheus.MetricName applies the exact sanitization the Prometheus
//     adapter uses when registering it.
//
// zprometheus.MetricName sanitizes every character invalid in a metric name to
// '_' (so a hyphenated workload name is stored as e.g. account_analyzer, not
// account-analyzer), guaranteeing this query targets the exact series the
// adapter emits. The result is matched on __name__ so it stays correct even if
// that sanitized name ever contains a character a bare selector would reject.
func ZAMQPUtilizationStat(e env.Env, namespace, name string) string {
	metric := zprometheus.MetricName(
		zamqp.ConsumerUtilizationStatName(podspec.StatsPrefix(e, namespace, name)),
	)

	return fmt.Sprintf("avg({__name__=%q})", metric)
}
