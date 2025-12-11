package zormsql

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zelement/zelem"
	"github.com/milagre/zote/go/zreflect"
	"github.com/milagre/zote/go/zsql/zsqlite3"
)

func TestSortVisitor(t *testing.T) {
	type testCase struct {
		table             table
		columnAliasPrefix string
		expected          string
	}

	for name, tc := range map[string]testCase{
		"TableAlias": {
			table:    table{name: "table", alias: "target"},
			expected: `"target"."id" ASC`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := &Config{
				mappings: map[string]Mapping{
					zreflect.TypeID(reflect.TypeOf(objectMapping.PtrType)): objectMapping,
				},
			}
			sv := sortVisitor{
				driver:            zsqlite3.Driver,
				mapping:           objectMapping,
				table:             tc.table,
				columnAliasPrefix: tc.columnAliasPrefix,
				cfg:               cfg,
			}

			part, values, err := sv.Visit(zelem.Asc(zelem.Field("ID")))
			require.NoError(t, err)

			assert.Equal(t, part, tc.expected)
			assert.Len(t, values, 0)
		})
	}
}

func TestSortVisitor_DotDelimitedField(t *testing.T) {
	type user struct {
		ID      string
		Name    string
		Address *address
	}

	type address struct {
		ID    string
		State string
		City  string
	}

	userMapping := Mapping{
		PtrType:    &user{},
		Table:      "users",
		PrimaryKey: []string{"id"},
		Columns: []Column{
			{Name: "id", Field: "ID"},
			{Name: "name", Field: "Name"},
		},
		Relations: []Relation{
			{
				Table: "addresses",
				Columns: map[string]string{
					"id": "address_id",
				},
				Field: "Address",
			},
		},
	}

	addressMapping := Mapping{
		PtrType:    &address{},
		Table:      "addresses",
		PrimaryKey: []string{"id"},
		Columns: []Column{
			{Name: "id", Field: "ID"},
			{Name: "state", Field: "State"},
			{Name: "city", Field: "City"},
		},
		Relations: []Relation{},
	}

	cfg := &Config{
		mappings: map[string]Mapping{
			zreflect.TypeID(reflect.TypeOf(&user{})):    userMapping,
			zreflect.TypeID(reflect.TypeOf(&address{})): addressMapping,
		},
	}

	t.Run("Single level relation", func(t *testing.T) {
		sv := sortVisitor{
			driver:  zsqlite3.Driver,
			mapping: userMapping,
			table:   table{name: "users", alias: "target"},
			cfg:     cfg,
		}

		part, values, err := sv.Visit(zelem.Asc(zelem.Field("Address.State")))
		require.NoError(t, err)

		assert.Equal(t, `"Address"."state" ASC`, part)
		assert.Len(t, values, 0)
	})

	t.Run("Descending sort", func(t *testing.T) {
		sv := sortVisitor{
			driver:  zsqlite3.Driver,
			mapping: userMapping,
			table:   table{name: "users", alias: "target"},
			cfg:     cfg,
		}

		part, values, err := sv.Visit(zelem.Desc(zelem.Field("Address.City")))
		require.NoError(t, err)

		assert.Equal(t, `"Address"."city" DESC`, part)
		assert.Len(t, values, 0)
	})

	t.Run("Missing relation", func(t *testing.T) {
		userMappingNoRelation := Mapping{
			PtrType:    &user{},
			Table:      "users",
			PrimaryKey: []string{"id"},
			Columns: []Column{
				{Name: "id", Field: "ID"},
			},
			Relations: []Relation{},
		}

		sv := sortVisitor{
			driver:  zsqlite3.Driver,
			mapping: userMappingNoRelation,
			table:   table{name: "users", alias: "target"},
			cfg:     cfg,
		}

		_, _, err := sv.Visit(zelem.Asc(zelem.Field("Address.State")))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "relation Address not found in mapping")
	})
}
