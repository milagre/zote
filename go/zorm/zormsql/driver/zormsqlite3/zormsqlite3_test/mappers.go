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
	UniqueKeys: [][]string{
		{
			"company",
		},
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
		{
			Name:  "contact_email",
			Field: "ContactEmail",
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

var UserAddressMapping = zormsql.Mapping{
	PtrType: &zormtest.UserAddress{},
	Table:   "user_addresses",
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
	Relations: []zormsql.Relation{},
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
			Name:  "user_address_id",
			Field: "AddressID",
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
				"user_address_id": "id",
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
