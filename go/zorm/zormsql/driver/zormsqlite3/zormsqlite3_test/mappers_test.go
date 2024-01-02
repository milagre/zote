package zormsqlite3_test

import (
	"github.com/milagre/zote/go/zorm/zormsql"
	"github.com/milagre/zote/go/zorm/zormtest"
)

var AccountMapping = zormsql.Mapping{
	PtrType: &zormtest.Account{},
	Table:   "accounts",
	PrimaryKey: []string{
		"id",
	},
	Columns: []zormsql.Column{
		{
			Name:     "id",
			Field:    "ID",
			NoInsert: true,
			NoUpdate: true,
		},
		{
			Name:     "created",
			Field:    "Created",
			NoInsert: true,
			NoUpdate: true,
		},
		{
			Name:     "modified",
			Field:    "Modified",
			NoInsert: true,
			NoUpdate: true,
		},
		{
			Name:  "company",
			Field: "Company",
		},
	},
}

var UserMapping = zormsql.Mapping{
	PtrType: &zormtest.User{},
	Table:   "users",
	PrimaryKey: []string{
		"id",
	},
	Columns: []zormsql.Column{
		{
			Name:     "id",
			Field:    "ID",
			NoInsert: true,
			NoUpdate: true,
		},
		{
			Name:     "created",
			Field:    "Created",
			NoInsert: true,
			NoUpdate: true,
		},
		{
			Name:     "modified",
			Field:    "Modified",
			NoInsert: true,
			NoUpdate: true,
		},
		{
			Name:  "account_id",
			Field: "AccountID",
		},
		{
			Name:  "first_name",
			Field: "FirstName",
		},
	},
	Relations: []zormsql.Relation{
		{
			Table: "accounts",
			Columns: map[string]string{
				"account_id": "id",
			},
			Field: "Account",
		},
	},
}
