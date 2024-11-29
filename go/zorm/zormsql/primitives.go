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
	table table

	columns []column
	fields  []string
	target  []interface{}

	primaryKey       []column
	primaryKeyFields []string
	primaryKeyTarget []interface{}

	relations       []string
	toOneRelations  map[string]joinStructure
	toManyRelations map[string]joinStructure
}

func (s structure) fullColumns() []column {
	result := []column{}
	result = append(result, s.primaryKey...)
	result = append(result, s.columns...)
	for _, f := range s.relations {
		if r, ok := s.toOneRelations[f]; ok {
			result = append(result, r.structure.fullColumns()...)
		} else if r, ok := s.toManyRelations[f]; ok {
			result = append(result, r.structure.fullColumns()...)
		}
	}
	return result
}

func (s structure) fullTarget() []interface{} {
	result := []interface{}{}
	result = append(result, s.primaryKeyTarget...)
	result = append(result, s.target...)
	for _, f := range s.relations {
		if r, ok := s.toOneRelations[f]; ok {
			result = append(result, r.structure.fullTarget()...)
		} else if r, ok := s.toManyRelations[f]; ok {
			result = append(result, r.structure.fullTarget()...)
		}
	}
	return result
}

func (s structure) getRelation(name string) (joinStructure, bool) {
	res, ok := s.toOneRelations[name]
	if !ok {
		res, ok = s.toManyRelations[name]
	}
	return res, ok
}

type joinStructure struct {
	onPairs   [][2]column
	onWhere   zclause.Clause
	structure structure
}
