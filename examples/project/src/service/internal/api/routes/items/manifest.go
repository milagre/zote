package items

import (
	"github.com/milagre/zote/go/zamqp"
	"github.com/milagre/zote/go/zapi"
	"github.com/milagre/zote/go/zorm"
)

// Routes returns item routes in mount order (collection before /items/{item_id}).
func Routes(repo zorm.Repository, pub zamqp.Publisher) []zapi.Route {
	return []zapi.Route{
		newList(repo),
		newItem(repo, pub),
	}
}
