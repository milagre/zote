package zormsql

import "time"

type Object struct {
	ID       string
	Created  time.Time
	Modified *time.Time

	Name string
}

var objectMapping Mapping = Mapping{
	PtrType:    &Object{},
	Table:      "objects",
	PrimaryKey: []string{"id"},
	Columns: []Column{
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
			Name:  "name",
			Field: "Name",
		},
	},
}
