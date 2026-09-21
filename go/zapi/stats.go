package zapi

import (
	"net/http"
	"sync"
	"time"

	"github.com/milagre/zote/go/zstats"
)

const (
	// StatsPrefix is the zstats prefix a server applies to its metrics,
	// composed onto the ambient process stats prefix.
	StatsPrefix = "zapi"

	// BusySecondsMetric is the counter of request-seconds a server has spent
	// with requests in flight: each second adds the number of requests open
	// during it. Its rate over a window is the mean concurrency across that
	// window, and counts requests much shorter than the scrape interval in full.
	BusySecondsMetric = "busy_seconds"
)

const routeTagUnknown = "unknown"

// BusySecondsStatName returns the fully-qualified zstats name a server
// publishes for [BusySecondsMetric], given the ambient process stats prefix
// (the value of the <PREFIX>_STATS_PREFIX env var, e.g. "app.apps.api").
// This is the name before any adapter-specific sanitization (see
// zprometheus.MetricName).
func BusySecondsStatName(statsPrefix string) string {
	return zstats.Qualify(zstats.Qualify(statsPrefix, StatsPrefix), BusySecondsMetric)
}

// requestStatTags holds the zstats tags used for zapi HTTP metrics so every
// observation (including deferred response counts) shares one stable label schema. Extend this
// struct and update tags()/tagsWithMatchedRoute together when adding a dimension.
type requestStatTags struct {
	Method string
	Path   string
	Route  string
}

func newRequestStatTags(r *http.Request) requestStatTags {
	return requestStatTags{
		Method: r.Method,
		Path:   r.URL.Path,
		Route:  routeTagUnknown,
	}
}

// tags returns stat tags before a route is resolved (e.g. no matching handler).
func (t requestStatTags) tags() zstats.Tags {
	return zstats.Tags{
		"method": t.Method,
		"path":   t.Path,
		"route":  t.Route,
	}
}

// tagsWithMatchedRoute returns tags after routing; path and route name come from the matched Route.
func (t requestStatTags) tagsWithMatchedRoute(route Route) zstats.Tags {
	return zstats.Tags{
		"method": t.Method,
		"path":   route.Path(),
		"route":  route.Name(),
	}
}

// busyClock integrates the number of requests in flight over time. The zero
// value is ready to use.
type busyClock struct {
	mu       sync.Mutex
	inFlight int64
	busy     time.Duration // accrued up to since
	since    time.Time
}

func (c *busyClock) add(now time.Time, delta int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.busy = c.accrued(now)
	c.since = now
	c.inFlight += delta
}

func (c *busyClock) current() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.inFlight
}

// total returns busy time through now, including requests still open.
func (c *busyClock) total(now time.Time) time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.accrued(now)
}

func (c *busyClock) accrued(now time.Time) time.Duration {
	if c.inFlight == 0 {
		return c.busy
	}

	return c.busy + time.Duration(c.inFlight)*now.Sub(c.since)
}
