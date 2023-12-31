package zormsqlite3_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/milagre/zote/go/zorm"
	"github.com/milagre/zote/go/zorm/zormsql"
	"github.com/milagre/zote/go/zsql/zsqlite3"
)

func setup(t *testing.T, cb func(context.Context, zorm.Repository)) {
	t.Helper()

	sourcedb, err := os.ReadFile("test.db")
	require.NoError(t, err, "reading test file")

	dir := os.TempDir()
	tempfile, err := os.CreateTemp(dir, fmt.Sprintf("zote_sqlite3_test-test.%s.*.db", t.Name()))
	require.NoError(t, err, "database temp file")

	tempfilename := tempfile.Name()
	defer os.Remove(tempfilename)

	_, err = tempfile.Write(sourcedb)
	require.NoError(t, err)
	tempfile.Close()

	conn, err := zsqlite3.Open(zsqlite3.FileConnectionString(tempfilename, zsqlite3.DefaultOptions()), 10)
	require.NoError(t, err, "opening database")
	defer conn.Close()

	repo := zormsql.NewRepository("test.db", conn)
	repo.AddMapping(AccountMapping)
	repo.AddMapping(UserMapping)

	cb(context.Background(), repo)
}
