package zsqlite3

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"

	"github.com/milagre/zote/go/zsql"
)

var Driver zsql.Driver = driver{}

type driver struct {
}

func (d driver) Name() string {
	return "sqlite3"
}

func (d driver) EscapeTable(t string) string {
	return "\"" + strings.ReplaceAll(t, "\"", "\\\"") + "\""
}

func (d driver) EscapeColumn(c string) string {
	return "\"" + strings.ReplaceAll(c, "\"", "\\\"") + "\""
}

func (d driver) EscapeTableColumn(t string, c string) string {
	return d.EscapeTable(t) + "." + d.EscapeColumn(c)
}

func (d driver) NullSafeEqualityOperator() string {
	return "IS"
}

func (d driver) IsConflictError(err error) bool {
	var sqliteErr *sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
}

func DefaultOptions() zsql.Options {
	return zsql.Options{
		"cache":         "shared",
		"_foreign_keys": "on",
	}
}

func FileConnectionString(filepath string, opts zsql.Options) string {
	return connectionString(filepath, opts)
}

func MemoryConnectionString(opts zsql.Options) string {
	return connectionString(":memory:", opts)
}

func connectionString(path string, opts zsql.Options) string {
	if opts == nil {
		opts = DefaultOptions()
	}

	params := url.Values{}
	for k, v := range opts {
		params[k] = []string{fmt.Sprintf("%s", v)}
	}

	return path + "?" + params.Encode()
}

func Open(dsn string, poolSize int) (zsql.Connection, error) {
	pool, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return zsql.Connection{}, fmt.Errorf("opening sqlite3 connection: %w", err)
	}

	pool.SetConnMaxIdleTime(5 * time.Minute)
	pool.SetMaxIdleConns((poolSize / 2) + 1)
	pool.SetMaxOpenConns(poolSize)

	return zsql.NewConnection(pool, Driver), nil
}
