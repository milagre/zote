package sqlite3

import (
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/mattn/go-sqlite3"

	zotesql "github.com/milagre/zote/go/sql"
)

var driver zotesql.Driver

func init() {
	driver = zotesql.NewDriver("sqlite3")
}

func DefaultOptions() zotesql.Options {
	return zotesql.Options{
		"cache":         "shared",
		"_foreign_keys": "on",
	}
}

func FileConnectionString(filepath string, opts zotesql.Options) string {
	return connectionString(filepath, opts)
}

func MemoryConnectionString(opts zotesql.Options) string {
	return connectionString(":memory:", opts)
}

func connectionString(path string, opts zotesql.Options) string {
	if opts == nil {
		opts = DefaultOptions()
	}

	params := url.Values{}
	for k, v := range opts {
		params[k] = []string{fmt.Sprintf("%s", v)}
	}

	return path + params.Encode()
}

func Open(dsn string, poolSize int) (*zotesql.Connection, error) {
	pool, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite3 connection: %w", err)
	}

	pool.SetConnMaxIdleTime(5 * time.Minute)
	pool.SetMaxIdleConns((poolSize / 2) + 1)
	pool.SetMaxOpenConns(poolSize)

	return zotesql.NewConnection(pool, driver), nil
}
