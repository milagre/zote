package zelem_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zelem"
	"github.com/milagre/zote/go/zelement/zsort"
)

func TestField(t *testing.T) {
	got := zelem.Field("test")
	want := zelement.Field{Name: "test"}
	assert.Equal(t, want, got)
}

func TestValue(t *testing.T) {
	got := zelem.Value(123)
	want := zelement.Value{Value: 123}
	assert.Equal(t, want, got)
}

func TestComparisons(t *testing.T) {
	left := zelem.Value(1)
	right := zelem.Value(2)

	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"Eq", zelem.Eq(left, right), zclause.Eq{Left: left, Right: right}},
		{"Gt", zelem.Gt(left, right), zclause.Gt{Left: left, Right: right}},
		{"Gte", zelem.Gte(left, right), zclause.Gte{Left: left, Right: right}},
		{"Lt", zelem.Lt(left, right), zclause.Lt{Left: left, Right: right}},
		{"Lte", zelem.Lte(left, right), zclause.Lte{Left: left, Right: right}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got)
		})
	}
}

func TestLogicalOperators(t *testing.T) {
	clause1 := zelem.Eq(zelem.Value(1), zelem.Value(1))
	clause2 := zelem.Eq(zelem.Value(2), zelem.Value(2))

	got := zelem.And(clause1, clause2)
	want := zclause.And{Clauses: []zclause.Clause{clause1, clause2}}
	assert.Equal(t, want, got)

	got2 := zelem.Or(clause1, clause2)
	want2 := zclause.Or{Clauses: []zclause.Clause{clause1, clause2}}
	assert.Equal(t, want2, got2)
}

func TestSorts(t *testing.T) {
	elem := zelem.Field("test")

	asc := zelem.Asc(elem)
	assert.Equal(t, zsort.Asc, asc.Direction)
	assert.Equal(t, elem, asc.Element)

	desc := zelem.Desc(elem)
	assert.Equal(t, zsort.Desc, desc.Direction)
	assert.Equal(t, elem, desc.Element)

	sorts := zelem.Sorts(asc, desc)
	assert.Len(t, sorts, 2)
	assert.Equal(t, asc, sorts[0])
	assert.Equal(t, desc, sorts[1])
}

func TestMethods(t *testing.T) {
	now := zelem.MethodNow()
	assert.Equal(t, "now", now.Name)
	assert.Empty(t, now.Params)

	match := zelem.MethodMatch("field", "search")
	assert.Equal(t, "match", match.Name)
	assert.Len(t, match.Params, 2)

	field, ok := match.Params[0].(zelement.Field)
	assert.True(t, ok)
	assert.Equal(t, "field", field.Name)

	value, ok := match.Params[1].(zelement.Value)
	assert.True(t, ok)
	assert.Equal(t, "search", value.Value)
}
