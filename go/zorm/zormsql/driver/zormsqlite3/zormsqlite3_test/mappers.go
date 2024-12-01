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
	Relations: []zormsql.Relation{
		{
			Table: "users",
			Columns: map[string]string{
				"id": "account_id",
			},
			Field: "Users",
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
		{
			Table: "user_auths",
			Columns: map[string]string{
				"id": "user_id",
			},
			Field: "Auths",
		},
		{
			Table: "user_addresses",
			Columns: map[string]string{
				"id": "user_id",
			},
			Field: "Address",
		},
	},
}

var UserAuthMapping = zormsql.Mapping{
	PtrType: &zormtest.UserAuth{},
	Table:   "user_auths",
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
			Name:  "user_id",
			Field: "UserID",
		},
		{
			Name:  "provider",
			Field: "Provider",
		},
		{
			Name:  "data",
			Field: "Data",
		},
	},
	Relations: []zormsql.Relation{
		{
			Table: "users",
			Columns: map[string]string{
				"user_id": "id",
			},
			Field: "User",
		},
	},
}

var UserAddressMapping = zormsql.Mapping{
	PtrType: &zormtest.UserAddress{},
	Table:   "user_addresses",
	PrimaryKey: []string{
		"user_id",
	},
	Columns: []zormsql.Column{
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
			Name:  "user_id",
			Field: "UserID",
		},
		{
			Name:  "street",
			Field: "Street",
		},
		{
			Name:  "city",
			Field: "City",
		},
		{
			Name:  "state",
			Field: "State",
		},
	},
	Relations: []zormsql.Relation{
		{
			Table: "users",
			Columns: map[string]string{
				"user_id": "id",
			},
			Field: "User",
		},
	},
}
