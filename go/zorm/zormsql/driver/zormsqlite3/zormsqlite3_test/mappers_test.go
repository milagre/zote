package zormsqlite3_test

import (
	"time"

	"github.com/milagre/zote/go/zorm/zormsql"
)

type Account struct {
	ID       string
	Created  time.Time
	Modified *time.Time

	Company string
}

type User struct {
	ID       string
	Created  time.Time
	Modified *time.Time

	AccountID string
	Account   *Account

	Name string
}

var AccountMapping = zormsql.Mapping{
	Type:  Account{},
	Table: "accounts",
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
	Type:  User{},
	Table: "users",
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
			Name:  "name",
			Field: "Name",
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
