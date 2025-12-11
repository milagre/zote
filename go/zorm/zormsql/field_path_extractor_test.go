package zormsql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zelement"
	"github.com/milagre/zote/go/zelement/zclause"
	"github.com/milagre/zote/go/zelement/zelem"
	"github.com/milagre/zote/go/zelement/zmethod"
	"github.com/milagre/zote/go/zelement/zsort"
)

func TestExtractFieldPaths(t *testing.T) {
	t.Run("Simple field", func(t *testing.T) {
		clause := zelem.Eq(zelem.Field("Name"), zelem.Value("John"))
		paths := extractFieldPaths(clause)

		require.Len(t, paths, 1)
		assert.Equal(t, "Name", paths[0])
	})

	t.Run("Dot-delimited field", func(t *testing.T) {
		clause := zelem.Eq(zelem.Field("Users.Address.State"), zelem.Value("PA"))
		paths := extractFieldPaths(clause)

		require.Len(t, paths, 1)
		assert.Equal(t, "Users.Address.State", paths[0])
	})

	t.Run("Multiple fields in AND clause", func(t *testing.T) {
		clause := zelem.And(
			zelem.Eq(zelem.Field("Name"), zelem.Value("John")),
			zelem.Eq(zelem.Field("Users.Address.State"), zelem.Value("PA")),
		)
		paths := extractFieldPaths(clause)

		require.Len(t, paths, 2)
		assert.Contains(t, paths, "Name")
		assert.Contains(t, paths, "Users.Address.State")
	})

	t.Run("Multiple fields in OR clause", func(t *testing.T) {
		clause := zelem.Or(
			zelem.Eq(zelem.Field("Name"), zelem.Value("John")),
			zelem.Eq(zelem.Field("Email"), zelem.Value("john@example.com")),
		)
		paths := extractFieldPaths(clause)

		require.Len(t, paths, 2)
		assert.Contains(t, paths, "Name")
		assert.Contains(t, paths, "Email")
	})

	t.Run("Nested clauses", func(t *testing.T) {
		clause := zelem.And(
			zelem.Eq(zelem.Field("Name"), zelem.Value("John")),
			zelem.Or(
				zelem.Eq(zelem.Field("Users.Address.State"), zelem.Value("PA")),
				zelem.Eq(zelem.Field("Users.Address.City"), zelem.Value("Philadelphia")),
			),
		)
		paths := extractFieldPaths(clause)

		require.Len(t, paths, 3)
		assert.Contains(t, paths, "Name")
		assert.Contains(t, paths, "Users.Address.State")
		assert.Contains(t, paths, "Users.Address.City")
	})

	t.Run("IN clause with multiple fields", func(t *testing.T) {
		clause := zclause.In{
			Left: []zelement.Element{
				zelem.Field("Users.Address.State"),
				zelem.Field("Users.Address.City"),
			},
			Right: [][]zelement.Element{
				{zelem.Value("PA"), zelem.Value("Philadelphia")},
			},
		}
		paths := extractFieldPaths(clause)

		require.Len(t, paths, 2)
		assert.Contains(t, paths, "Users.Address.State")
		assert.Contains(t, paths, "Users.Address.City")
	})

	t.Run("Method with field parameter", func(t *testing.T) {
		clause := zelem.Truthy(
			zmethod.NewContains(
				zelem.Field("Users.Address.State"),
				zelem.Value("PA"),
			),
		)
		paths := extractFieldPaths(clause)

		require.Len(t, paths, 1)
		assert.Contains(t, paths, "Users.Address.State")
	})

	t.Run("NOT clause", func(t *testing.T) {
		clause := zclause.Not{
			Clause: zelem.Eq(zelem.Field("Users.Address.State"), zelem.Value("PA")),
		}
		paths := extractFieldPaths(clause)

		require.Len(t, paths, 1)
		assert.Contains(t, paths, "Users.Address.State")
	})

	t.Run("Comparison operators", func(t *testing.T) {
		tests := []struct {
			name   string
			clause zclause.Clause
		}{
			{"Gt", zelem.Gt(zelem.Field("Users.Address.State"), zelem.Value("M"))},
			{"Gte", zelem.Gte(zelem.Field("Users.Address.State"), zelem.Value("M"))},
			{"Lt", zelem.Lt(zelem.Field("Users.Address.State"), zelem.Value("Z"))},
			{"Lte", zelem.Lte(zelem.Field("Users.Address.State"), zelem.Value("Z"))},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				paths := extractFieldPaths(tt.clause)
				require.Len(t, paths, 1)
				assert.Contains(t, paths, "Users.Address.State")
			})
		}
	})
}

func TestExtractFieldPathsFromSorts(t *testing.T) {
	t.Run("Simple field sort", func(t *testing.T) {
		sorts := []zsort.Sort{
			zelem.Asc(zelem.Field("Name")),
		}
		paths := extractFieldPathsFromSorts(sorts)

		require.Len(t, paths, 1)
		assert.Equal(t, "Name", paths[0])
	})

	t.Run("Dot-delimited field sort", func(t *testing.T) {
		sorts := []zsort.Sort{
			zelem.Asc(zelem.Field("Users.Address.State")),
		}
		paths := extractFieldPathsFromSorts(sorts)

		require.Len(t, paths, 1)
		assert.Equal(t, "Users.Address.State", paths[0])
	})

	t.Run("Multiple sorts", func(t *testing.T) {
		sorts := []zsort.Sort{
			zelem.Asc(zelem.Field("Name")),
			zelem.Desc(zelem.Field("Users.Address.State")),
			zelem.Asc(zelem.Field("Users.Address.City")),
		}
		paths := extractFieldPathsFromSorts(sorts)

		require.Len(t, paths, 3)
		assert.Contains(t, paths, "Name")
		assert.Contains(t, paths, "Users.Address.State")
		assert.Contains(t, paths, "Users.Address.City")
	})

	t.Run("Empty sorts", func(t *testing.T) {
		sorts := []zsort.Sort{}
		paths := extractFieldPathsFromSorts(sorts)

		require.Len(t, paths, 0)
	})
}
