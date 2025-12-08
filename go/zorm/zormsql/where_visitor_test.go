package zormsql

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/milagre/zote/go/zelement/zelem"
	"github.com/milagre/zote/go/zelement/zmethod"
	"github.com/milagre/zote/go/zsql/zmysql"
)

type user struct {
	ID   int
	Name string
	Age  int
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
