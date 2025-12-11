package zormsql

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zelement/zelem"
	"github.com/milagre/zote/go/zelement/zmethod"
	"github.com/milagre/zote/go/zreflect"
	"github.com/milagre/zote/go/zsql/zmysql"
)

type user struct {
	ID      int
	Name    string
	Age     int
	Address *address
}

type address struct {
	ID    string
	State string
	City  string
}

var mapping = Mapping{
	PtrType:    &user{},
	Table:      "users",
	PrimaryKey: []string{"id"},
	Columns: []Column{
		{Name: "id", Field: "ID", NoInsert: true, NoUpdate: true},
		{Name: "name", Field: "Name"},
		{Name: "age", Field: "Age"},
	},
	Relations: []Relation{},
}

var addressMapping = Mapping{
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

var mappingWithAddress = Mapping{
	PtrType:    &user{},
	Table:      "users",
	PrimaryKey: []string{"id"},
	Columns: []Column{
		{Name: "id", Field: "ID", NoInsert: true, NoUpdate: true},
		{Name: "name", Field: "Name"},
		{Name: "age", Field: "Age"},
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

func TestWhereVisitor(t *testing.T) {
	driver := zmysql.Driver
	visitor := whereVisitor{driver: driver, mapping: mapping, table: table{name: mapping.Table}, columnAliasPrefix: ""}
	where, values, err := visitor.Visit(zelem.Eq(zelem.Field("Name"), zelem.Value("John")))

	assert.NoError(t, err)
	assert.Equal(t, "`name` <=> ?", where)
	assert.Equal(t, []interface{}{"John"}, values)
}

func TestWhereVisitor_ComplexClause(t *testing.T) {
	driver := zmysql.Driver
	visitor := whereVisitor{driver: driver, mapping: mapping, table: table{name: mapping.Table}, columnAliasPrefix: ""}
	where, values, err := visitor.Visit(
		zelem.And(
			zelem.Or(
				zelem.Eq(zelem.Field("Name"), zelem.Value("John")),
				zelem.Eq(zelem.Field("Name"), zelem.Value("Jane")),
			),
			zelem.Gt(
				zelem.Field("Age"),
				zelem.Value(20),
			),
		),
	)

	assert.NoError(t, err)
	assert.Equal(t, "((`name` <=> ? OR `name` <=> ?) AND `age` > ?)", where)
	assert.Equal(t, []interface{}{
		"John",
		"Jane",
		20,
	}, values)
}

func TestWhereVisitor_Now(t *testing.T) {
	driver := zmysql.Driver
	visitor := whereVisitor{driver: driver, mapping: mapping, table: table{name: mapping.Table}, columnAliasPrefix: ""}
	where, values, err := visitor.Visit(zelem.Eq(zelem.MethodNow(), zelem.Value(true)))

	assert.NoError(t, err)
	assert.Equal(t, "now() <=> ?", where)
	assert.Equal(t, []interface{}{true}, values)
}

func TestWhereVisitor_ComplexMethod(t *testing.T) {
	driver := zmysql.Driver
	visitor := whereVisitor{driver: driver, mapping: mapping, table: table{name: mapping.Table}, columnAliasPrefix: ""}
	where, values, err := visitor.Visit(
		zelem.Truthy(
			zmethod.NewContains(
				zmethod.NewRegexpReplace(zelem.Field("Name"), zelem.Value("ohn"), zelem.Value("ane")),
				zelem.Value("Jane"),
			),
		),
	)

	assert.NoError(t, err)
	assert.Equal(t, "INSTR(regexp_replace(`name`, ?, ?), ?) > 0", where)
	assert.Equal(t, []interface{}{
		"ohn",
		"ane",
		"Jane",
	}, values)
}

func TestWhereVisitor_DotDelimitedField(t *testing.T) {
	driver := zmysql.Driver
	cfg := &Config{
		mappings: map[string]Mapping{
			zreflect.TypeID(reflect.TypeOf(&user{})):    mappingWithAddress,
			zreflect.TypeID(reflect.TypeOf(&address{})): addressMapping,
		},
	}

	visitor := whereVisitor{
		driver:  driver,
		mapping: mappingWithAddress,
		table:   table{name: mappingWithAddress.Table, alias: "target"},
		cfg:     cfg,
	}

	t.Run("Single level relation", func(t *testing.T) {
		where, values, err := visitor.Visit(zelem.Eq(zelem.Field("Address.State"), zelem.Value("PA")))

		require.NoError(t, err)
		assert.Equal(t, "`Address`.`state` <=> ?", where)
		assert.Equal(t, []interface{}{"PA"}, values)
	})

	t.Run("Dot-delimited in complex clause", func(t *testing.T) {
		where, values, err := visitor.Visit(
			zelem.And(
				zelem.Eq(zelem.Field("Name"), zelem.Value("John")),
				zelem.Eq(zelem.Field("Address.State"), zelem.Value("PA")),
			),
		)

		require.NoError(t, err)
		assert.Equal(t, "(`target`.`name` <=> ? AND `Address`.`state` <=> ?)", where)
		assert.Equal(t, []interface{}{"John", "PA"}, values)
	})

	t.Run("Dot-delimited in method", func(t *testing.T) {
		where, values, err := visitor.Visit(
			zelem.Truthy(
				zmethod.NewContains(
					zelem.Field("Address.State"),
					zelem.Value("PA"),
				),
			),
		)

		require.NoError(t, err)
		assert.Equal(t, "INSTR(`Address`.`state`, ?) > 0", where)
		assert.Equal(t, []interface{}{"PA"}, values)
	})

	t.Run("Missing relation", func(t *testing.T) {
		visitorNoRelation := whereVisitor{
			driver:  driver,
			mapping: mapping, // mapping without Address relation
			table:   table{name: mapping.Table, alias: "target"},
			cfg:     cfg,
		}

		_, _, err := visitorNoRelation.Visit(zelem.Eq(zelem.Field("Address.State"), zelem.Value("PA")))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "relation Address not found in mapping")
	})
}
