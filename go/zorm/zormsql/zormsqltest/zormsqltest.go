package zormsqltest

import (
	"testing"

	"github.com/milagre/zote/go/zorm/zormsql"
	"github.com/stretchr/testify/assert"
)

func ValidateMapping(t *testing.T, m zormsql.Mapping) {
	assert.NotZero(t, m.Table, "sql mapper validation failed: table not defined %+v", m)
	assert.NotZero(t, m.PtrType, "sql mapper validation failed for table %s: model pointer type not defined", m.Table)
	assert.NotZero(t, m.PrimaryKey, "sql mapper validation failed for type %T: primary key not defined", m.PtrType)
	assert.Greater(t, len(m.PrimaryKey), 0, "sql mapper validation failed for type %T: primary key not defined (no length)", m.PtrType)
	assert.NotZero(t, m.Columns, "sql mapper validation failed for type %T: columns not defined", m.PtrType)
	assert.Greater(t, len(m.Columns), 0, "sql mapper validation failed for type %T: columns not defined (no length)", m.PtrType)
}
