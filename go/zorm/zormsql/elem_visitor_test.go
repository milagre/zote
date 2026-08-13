package zormsql

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zelement/zelem"
	"github.com/milagre/zote/go/zelement/zmethod"
	"github.com/milagre/zote/go/zreflect"
	"github.com/milagre/zote/go/zsql/zsqlite3"
)

func newObjectElemVisitor() elemVisitor {
	cfg := &Config{
		mappings: map[string]Mapping{
			zreflect.TypeID(reflect.TypeOf(objectMapping.PtrType)): objectMapping,
		},
	}

	return elemVisitor{
		driver:  zsqlite3.Driver,
		mapping: objectMapping,
		table:   table{name: "objects", alias: "target"},
		cfg:     cfg,
	}
}

func TestElemVisitor_Field(t *testing.T) {
	ev := newObjectElemVisitor()

	part, values, err := ev.Visit(zelem.Field("Name"))
	require.NoError(t, err)

	assert.Equal(t, `"target"."name"`, part)
	assert.Len(t, values, 0)
}

func TestElemVisitor_Value(t *testing.T) {
	ev := newObjectElemVisitor()

	part, values, err := ev.Visit(zelem.Value("bank"))
	require.NoError(t, err)

	assert.Equal(t, "?", part)
	assert.Equal(t, []interface{}{"bank"}, values)
}

func TestElemVisitor_UnmappedField(t *testing.T) {
	ev := newObjectElemVisitor()

	_, _, err := ev.Visit(zelem.Field("Missing"))
	require.Error(t, err)
}

// Methods the driver has no template for fall back to a plain SQL call, which
// is how coalesce reaches the database.
func TestElemVisitor_MethodWithoutDriverTemplate(t *testing.T) {
	ev := newObjectElemVisitor()

	part, values, err := ev.Visit(zmethod.NewCoalesce(
		zelem.Field("Name"),
		zelem.Value("unknown"),
	))
	require.NoError(t, err)

	assert.Equal(t, `coalesce("target"."name", ?)`, part)
	assert.Equal(t, []interface{}{"unknown"}, values)
}

func TestElemVisitor_NestedMethods(t *testing.T) {
	ev := newObjectElemVisitor()

	part, values, err := ev.Visit(zmethod.NewCoalesce(
		zmethod.NewCoalesce(zelem.Field("Name"), zelem.Value("inner")),
		zelem.Value("outer"),
	))
	require.NoError(t, err)

	assert.Equal(t, `coalesce(coalesce("target"."name", ?), ?)`, part)
	assert.Equal(t, []interface{}{"inner", "outer"}, values)
}

// Methods the driver does template are expanded with their parameters
// substituted into the template, in order.
func TestElemVisitor_MethodWithDriverTemplate(t *testing.T) {
	ev := newObjectElemVisitor()

	part, values, err := ev.Visit(zmethod.NewContains(
		zelem.Field("Name"),
		zelem.Value("chase"),
	))
	require.NoError(t, err)

	assert.Equal(t, `INSTR("target"."name", ?) > 0`, part)
	assert.Equal(t, []interface{}{"chase"}, values)
}

func TestElemVisitor_MethodWithDriverTemplateAndExtraParams(t *testing.T) {
	ev := newObjectElemVisitor()

	// The template only has two slots, so anything beyond them is appended -
	// the values still have to arrive in parameter order.
	part, values, err := ev.Visit(zelement.Method{
		Name: string(zmethod.Contains),
		Params: []zelement.Element{
			zelem.Field("Name"),
			zelem.Value("chase"),
			zelem.Value("extra"),
		},
	})
	require.NoError(t, err)

	assert.Equal(t, `INSTR("target"."name", ?) > 0?`, part)
	assert.Equal(t, []interface{}{"chase", "extra"}, values)
}

func TestElemVisitor_MethodPropagatesParamError(t *testing.T) {
	ev := newObjectElemVisitor()

	_, _, err := ev.Visit(zmethod.NewCoalesce(
		zelem.Field("Missing"),
		zelem.Value("fallback"),
	))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "param 0")
}

func TestElemVisitor_MethodOverDotDelimitedField(t *testing.T) {
	cfg := &Config{
		mappings: map[string]Mapping{
			zreflect.TypeID(reflect.TypeOf(&user{})):    mappingWithAddress,
			zreflect.TypeID(reflect.TypeOf(&address{})): addressMapping,
		},
	}

	ev := elemVisitor{
		driver:  zsqlite3.Driver,
		mapping: mappingWithAddress,
		table:   table{name: "users", alias: "target"},
		cfg:     cfg,
	}

	part, values, err := ev.Visit(zmethod.NewCoalesce(
		zelem.Field("Name"),
		zelem.Field("Address.City"),
	))
	require.NoError(t, err)

	assert.Equal(t, `coalesce("target"."name", "Address"."city")`, part)
	assert.Len(t, values, 0)
}

// Sorting by a method is what lets a curated column fall back to the value it
// mirrors without losing the ordering to the database.
func TestSortVisitor_Method(t *testing.T) {
	cfg := &Config{
		mappings: map[string]Mapping{
			zreflect.TypeID(reflect.TypeOf(objectMapping.PtrType)): objectMapping,
		},
	}

	sv := sortVisitor{
		driver:  zsqlite3.Driver,
		mapping: objectMapping,
		table:   table{name: "objects", alias: "target"},
		cfg:     cfg,
	}

	part, values, err := sv.Visit(zelem.Asc(zmethod.NewCoalesce(
		zelem.Field("Name"),
		zelem.Value("zzz"),
	)))
	require.NoError(t, err)

	assert.Equal(t, `coalesce("target"."name", ?) ASC`, part)
	assert.Equal(t, []interface{}{"zzz"}, values)
}
