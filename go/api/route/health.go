package route

import (
	"net/http"

	zoteapi "github.com/milagre/zote/go/api"
)

type health struct {
	path string
}

func NewHealth(path string) zoteapi.Route {
	return &health{
		path: path,
	}
}

func (r *health) Path() string {
	return r.path
}

func (r *health) Methods() zoteapi.Methods {
	return zoteapi.Methods{
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

func (r *health) health(req zoteapi.Request) zoteapi.ResponseBuilder {
	return zoteapi.Response200OK()
}
