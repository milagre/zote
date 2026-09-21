package deployment

import (
	"fmt"

	"github.com/milagre/zote/go/zapi"
	"github.com/milagre/zote/go/zstats/zprometheus"

	"github.com/milagre/zote/pulumi/env"
	"github.com/milagre/zote/pulumi/k8s/internal/podspec"
)

// ZAPIUtilizationStat returns the PromQL that reads a workload's mean in-flight
// request count as a per-replica utilization percentage, for use as
// [UtilizationTrigger.Query]. Only HTTP workloads running a zapi server expose
// this metric; a workload autoscaling on any other signal must supply its own
// query.
//
// e, namespace, and name identify the workload exactly as passed to New, so the
// query targets the metric name that workload emits. Both halves of the stored
// name come from the zote runtime rather than being restated here:
//
//   - zapi.BusySecondsStatName qualifies the ambient stats prefix with the
//     server prefix and metric the server emits, and
//   - zprometheus.MetricName applies the exact sanitization the Prometheus
//     adapter uses when registering it.
//
// perReplica is how many requests in flight one replica is sized to carry, and
// must be >= 1 (zero divides by zero in PromQL, which pins the workload to its
// replica ceiling). The runtime publishes busy request-seconds, whose rate is
// the mean number of requests in flight over the window; the division turns
// that into the 0-100 signal [UtilizationTrigger] expects: a replica holding
// perReplica requests on average reads as 100%. The window spans several
// scrapes so a single missed scrape does not blank the signal. Averaging rather than summing is
// what makes that per-replica — the trigger multiplies back up by the running
// replica count.
//
// The metric is matched on __name__ so the query stays correct even if the
// sanitized name contains a character a bare selector would reject.
func ZAPIUtilizationStat(e env.Env, namespace, name string, perReplica int) string {
	metric := zprometheus.MetricName(
		zapi.BusySecondsStatName(podspec.StatsPrefix(e, namespace, name)),
	)

	return fmt.Sprintf("avg(rate({__name__=%q}[1m])) * 100 / %d", metric, perReplica)
}
