package zormsql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zelement/zelem"
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
			sv := sortVisitor{
				driver:            zsqlite3.Driver,
				mapping:           objectMapping,
				table:             tc.table,
				columnAliasPrefix: tc.columnAliasPrefix,
			}

			part, values, err := sv.Visit(zelem.Asc(zelem.Field("ID")))
			require.NoError(t, err)

			assert.Equal(t, part, tc.expected)
			assert.Len(t, values, 0)
		})
	}
}
