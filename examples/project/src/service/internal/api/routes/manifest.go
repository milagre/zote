package routes

import (
	"github.com/milagre/zote/go/zamqp"
	"github.com/milagre/zote/go/zapi"
	"github.com/milagre/zote/go/zapi/zroute"
	"github.com/milagre/zote/go/zorm"

	"github.com/milagre/zote/examples/project/src/service/internal/api/routes/items"
)

// Routes returns zapi routes in mount order (root first, then tree paths).
func Routes(repo zorm.Repository, pub zamqp.Publisher) []zapi.Route {
	out := []zapi.Route{
		newRoot(),
		zroute.NewHealth("/_health"),
	}
	out = append(out, items.Routes(repo, pub)...)
	return out
}
