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
// The result matches on __name__ rather than naming the metric directly so the
// query parses regardless of the Prometheus name-validation scheme; workload
// names routinely contain hyphens, which are invalid in a bare metric name.
func ZAMQPUtilizationStat(e env.Env, namespace, name string) string {
	metric := zprometheus.MetricName(
		zamqp.ConsumerUtilizationStatName(podspec.StatsPrefix(e, namespace, name)),
	)

	return fmt.Sprintf("avg({__name__=%q})", metric)
}
