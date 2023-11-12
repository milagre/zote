package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"

	zotesql "github.com/milagre/zote/go/sql"
)

var driver zotesql.Driver

func init() {
	driver = zotesql.NewDriver("mysql")
}

func DefaultOptions() zotesql.Options {
	return zotesql.Options{
		"collation": "utf8mb4_bin",
		"charset":   "utf8mb4",
		"parseTime": true,
		"timeout":   5 * time.Second, // Connection timeout only
	}
}

func TCPConnectionString(user string, pass string, host string, port int, db string, opts zotesql.Options) string {
	if opts == nil {
		opts = DefaultOptions()
	}

	c := mysql.Config{
		User:   user,
		Passwd: pass,
		Addr:   fmt.Sprintf("%s:%d", host, port),
		DBName: db,
		Params: opts.ToStringMapString(),
		Net:    "tcp",
	}

	return c.FormatDSN()
}

func Open(dsn string, poolSize int) (*zotesql.Connection, error) {
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening mysql connection: %w", err)
	}

	pool.SetConnMaxIdleTime(5 * time.Minute)
	pool.SetMaxIdleConns((poolSize / 2) + 1)
	pool.SetMaxOpenConns(poolSize)

	return zotesql.NewConnection(pool, driver), nil
}
