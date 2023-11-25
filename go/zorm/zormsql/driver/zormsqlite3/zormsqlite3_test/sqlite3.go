package zormsqlite3_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zorm"
	"github.com/milagre/zote/go/zorm/zormsql"
	"github.com/milagre/zote/go/zsql"
	"github.com/milagre/zote/go/zsql/zsqlite3"

	zormsqlite3_test_mappers "github.com/milagre/zote/go/zorm/zormsql/driver/zormsqlite3/zormsqlite3_test/mappers"
)

func setup(t *testing.T, cb func(*zsql.Connection, zorm.Repository)) {
	t.Helper()

	f, err := os.CreateTemp("zote_sqlite3_test", fmt.Sprintf("test.%s.*.db", t.Name()))
	require.NoError(t, err, "database temp file")
	f.Close()
	defer os.Remove(f.Name())

	conn, err := zsqlite3.Open(zsqlite3.FileConnectionString(f.Name(), nil), 10)
	require.NoError(t, err, "opening database")
	defer conn.Close()

	source := zormsql.NewSource("test.db", conn)

	mappers := zormsqlite3_test_mappers.Mappers(source)

	r := zormsql.New(mappers)

	cb(conn, r)
}
