package zsqlite3

import (
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/milagre/zote/go/zsql"
)

var driver zsql.Driver

func init() {
	driver = zsql.NewDriver("sqlite3")
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

	return path + params.Encode()
}

func Open(dsn string, poolSize int) (zsql.Connection, error) {
	pool, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return zsql.Connection{}, fmt.Errorf("opening sqlite3 connection: %w", err)
	}

	pool.SetConnMaxIdleTime(5 * time.Minute)
	pool.SetMaxIdleConns((poolSize / 2) + 1)
	pool.SetMaxOpenConns(poolSize)

	return zsql.NewConnection(pool, driver), nil
}
