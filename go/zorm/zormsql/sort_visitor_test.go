package zormsql

import (
	"testing"

	"github.com/milagre/zote/go/zelement/zelem"
	"github.com/milagre/zote/go/zsql/zsqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSortVisitor(t *testing.T) {
	type testCase struct {
		tableAlias        string
		columnAliasPrefix string
		expected          string
	}

	for name, tc := range map[string]testCase{
		"Simple": {
			expected: `"id" ASC`,
		},
		"TableAlias": {
			tableAlias: "target",
			expected:   `"target"."id" ASC`,
		},
		"ColumnAliasPrefix": {
			columnAliasPrefix: "foreign",
			expected:          `"foreign_id" ASC`,
		},
		"TableAliasColumnAliasPrefix": {
			tableAlias:        "target",
			columnAliasPrefix: "foreign",
			expected:          `"target"."foreign_id" ASC`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			sv := sortVisitor{
				driver:            zsqlite3.Driver,
				mapping:           objectMapping,
				tableAlias:        tc.tableAlias,
				columnAliasPrefix: tc.columnAliasPrefix,
			}

			part, values, err := sv.Visit(zelem.Asc(zelem.Field("ID")))
			require.NoError(t, err)

			assert.Equal(t, part, tc.expected)
			assert.Len(t, values, 0)
		})
	}
}
