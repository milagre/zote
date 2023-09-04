package route

import (
	"net/http"

	"github.com/milagre/zote/go/api"
)

type health struct {
	path string
}

func NewHealth(path string) api.Route {
	return &health{
		path: path,
	}
}

func (r *health) Name() string {
	return "_health"
}

func (r *health) Path() string {
	return r.path
}

func (r *health) Methods() api.Methods {
	return api.Methods{
		http.MethodGet: {
			Handler: r.health,
		},
		http.MethodHead: {
			Handler: r.health,
		},
		http.MethodOptions: {
			Handler: r.health,
		},
	}
}

func (r *health) health(req api.Request) api.ResponseBuilder {
	return api.Response200OK()
}
