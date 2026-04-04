package models

import (
	"time"

	"github.com/milagre/zote/go/zorm/zormsql"
)

// Item is the relational / ORM shape for the items table (not used directly as JSON).
type Item struct {
	ID       string
	Created  time.Time
	Modified *time.Time
	Name     string
	Deleted  bool
}

var ItemMapping = zormsql.Mapping{
	PtrType:    &Item{},
	Table:      "items",
	PrimaryKey: []string{"id"},
	Columns: []zormsql.Column{
		{Name: "id", Field: "ID"},
		{Name: "created", Field: "Created"},
		{Name: "modified", Field: "Modified"},
		{Name: "name", Field: "Name"},
		{Name: "deleted", Field: "Deleted"},
	},
}

func AddMappings(r *zormsql.Repository) {
	r.AddMapping(ItemMapping)
}
