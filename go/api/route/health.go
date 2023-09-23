package route

import (
	"encoding/json"
	"net/http"

	"github.com/milagre/zote/go/api"
	"github.com/milagre/zote/go/build"
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

type result struct {
	Version string `json:"version"`
}

func (r *health) health(req api.Request) api.ResponseBuilder {
	result := result{
		Version: build.Version(),
	}
	b, _ := json.Marshal(result)
	return api.BasicResponse(
		http.StatusOK,
		http.Header{},
		b,
	)
}
