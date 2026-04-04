package routes

import (
	"encoding/json"
	"net/http"

	"github.com/milagre/zote/go/zapi"
	"github.com/milagre/zote/go/zlog"

	"github.com/milagre/zote/examples/project/src/service/internal/api/names"
)

type rootRoute struct{}

// newRoot serves GET / with a tiny JSON payload.
func newRoot() zapi.Route { return &rootRoute{} }

func (*rootRoute) Name() string { return names.Root }

func (*rootRoute) Path() string { return "" }

func (*rootRoute) Methods() zapi.Methods {
	return zapi.Methods{
		http.MethodGet: {Handler: getRoot},
	}
}

func getRoot(req zapi.Request) zapi.ResponseBuilder {
	body, err := json.Marshal(map[string]string{"status": "ok"})
	if err != nil {
		zlog.FromContext(req.Context()).Errorf("marshal root: %v", err)
		return zapi.Response500InternalServerError()
	}
	h := http.Header{}
	h.Set("Content-Type", "application/json; charset=utf-8")
	return zapi.BasicResponse(http.StatusOK, h, body)
}
