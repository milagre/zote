package zormsql

import (
	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zsql"
)

type table struct {
	name  string
	alias string
}

func (t table) escaped(d zsql.Driver) string {
	return d.EscapeTable(t.name)
}

func (t table) escapedAlias(d zsql.Driver) string {
	if t.alias == "" {
		return t.escaped(d)
	}
	return d.EscapeTable(t.alias)
}

type column struct {
	table table
	name  string
	alias string
}

func (c column) escaped(d zsql.Driver) string {
	if c.table.alias == "" {
		return d.EscapeColumn(c.name)
	}
	return d.EscapeTableColumn(c.table.alias, c.name)
}

func (c column) escapedAlias(d zsql.Driver) string {
	if c.alias == "" {
		return c.escaped(d)
	}
	if c.table.alias == "" {
		return d.EscapeColumn(c.alias)
	}
	return d.EscapeTableColumn(c.table.alias, c.alias)
}

type join struct {
	leftTable  table
	rightTable table
	onPairs    [][2]column
	onWhere    zclause.Clause
}

type structure struct {
	columns []column
	fields  []string
	target  []interface{}

	relations       []string
	toOneRelations  map[string]structure
	toManyRelations map[string]structure
}
