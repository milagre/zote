package zamqp

import "github.com/milagre/zote/go/zstats"

const (
	// ConsumerStatsPrefix is the zstats prefix a direct consumer applies to its
	// metrics, composed onto the ambient process stats prefix.
	ConsumerStatsPrefix = "zamqp.consumer"

	// ConsumerUtilizationMetric is the gauge (0-100) a direct consumer reports
	// for busy workers over configured concurrency.
	ConsumerUtilizationMetric = "utilization"
)

// ConsumerUtilizationStatName returns the fully-qualified zstats name a direct
// consumer publishes for utilization, given the ambient process stats prefix
// (the value of the <PREFIX>_STATS_PREFIX env var, e.g.
// "app.apps.worker"). This is the name before any adapter-specific
// sanitization (see zprometheus.MetricName).
func ConsumerUtilizationStatName(statsPrefix string) string {
	return zstats.Qualify(zstats.Qualify(statsPrefix, ConsumerStatsPrefix), ConsumerUtilizationMetric)
}
