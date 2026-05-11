package zapi

import (
	"net/http"

	"github.com/milagre/zote/go/zstats"
)

const routeTagUnknown = "unknown"

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
