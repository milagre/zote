package sqlite3_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	zoteorm "github.com/milagre/zote/go/orm"
	zoteormsql "github.com/milagre/zote/go/orm/sql"
	zotesql "github.com/milagre/zote/go/sql"
	zotesqlite3 "github.com/milagre/zote/go/sql/sqlite3"

	sqlite3_test_mappers "github.com/milagre/zote/go/orm/sql/driver/sqlite3/sqlite3_test/mappers"
)

func setup(t *testing.T, cb func(*zotesql.Connection, zoteorm.Repository)) {
	t.Helper()

	f, err := os.CreateTemp("zote_sqlite3_test", fmt.Sprintf("test.%s.*.db", t.Name()))
	require.NoError(t, err, "database temp file")
	f.Close()
	defer os.Remove(f.Name())

	conn, err := zotesqlite3.Open(zotesqlite3.FileConnectionString(f.Name(), nil), 10)
	require.NoError(t, err, "opening database")
	defer conn.Close()

	source := zoteormsql.NewSource("test.db", conn)

	mappers := sqlite3_test_mappers.Mappers(source)

	r := zoteormsql.New(mappers)

	cb(conn, r)
}
