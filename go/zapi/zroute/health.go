package zroute

import (
	"encoding/json"
	"net/http"

	"github.com/milagre/zote/go/zapi"
	"github.com/milagre/zote/go/zbuild"
)

type health struct {
	path string
}

func NewHealth(path string) zapi.Route {
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

func (r *health) Methods() zapi.Methods {
	return zapi.Methods{
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

func (r *health) health(req zapi.Request) zapi.ResponseBuilder {
	result := result{
		Version: zbuild.Version(),
	}
	b, _ := json.Marshal(result)
	return zapi.BasicResponse(
		http.StatusOK,
		http.Header{},
		b,
	)
}
