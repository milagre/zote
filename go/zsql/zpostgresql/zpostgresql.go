package zpostgresql

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/milagre/zote/go/zelement/zmethod"
	"github.com/milagre/zote/go/zsql"
)

var Driver zsql.Driver = driver{}

type driver struct{}

func (d driver) Name() string {
	return "postgresql"
}

func (d driver) EscapeTable(t string) string {
	return EscapeIdentifier(t)
}

func (d driver) EscapeColumn(c string) string {
	return EscapeIdentifier(c)
}

func (d driver) EscapeTableColumn(t string, c string) string {
	return EscapeIdentifier(t) + "." + EscapeIdentifier(c)
}

func (d driver) NullSafeEqualityOperator() string {
	// Performant in 18+, not in 17-
	return "IS NOT DISTINCT FROM"
}

func (d driver) PrepareMethod(m string) *string {
	var result *string

	switch zmethod.Method(m) {
	case zmethod.Contains:
		v := "POSITION(%s IN %s) > 0"
		result = &v
	}

	return result
}

func (d driver) IsConflictError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func DefaultOptions() zsql.Options {
	return zsql.Options{
		"connect_timeout": 5 * time.Second, // Connection timeout only
		"sslmode":         "prefer",
	}
}

func TCPConnectionString(user string, pass string, host string, port int, db string, opts zsql.Options) string {
	if opts == nil {
		opts = DefaultOptions()
	}

	var connect_timeout time.Duration
	if to, ok := opts["connect_timeout"]; !ok {
		connect_timeout = 5 * time.Second
	} else if tod, ok := to.(time.Duration); !ok {
		connect_timeout = 5 * time.Second
	} else {
		connect_timeout = tod
	}

	c := pgx.ConnConfig{
		Config: pgconn.Config{
			Host:           host,
			Port:           uint16(port),
			Database:       db,
			User:           user,
			Password:       pass,
			ConnectTimeout: connect_timeout,
			RuntimeParams:  opts.ToStringMapString(),
		},
	}

	return c.ConnString()
}

func Open(dsn string, poolSize int) (zsql.Connection, error) {
	pool, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening postgresql connection: %w", err)
	}

	pool.SetConnMaxIdleTime(5 * time.Minute)
	pool.SetMaxIdleConns((poolSize / 2) + 1)
	pool.SetMaxOpenConns(poolSize)

	return zsql.NewConnection(pool, Driver), nil
}

func EscapeIdentifier(identifier string) string {
	return "\"" + strings.ReplaceAll(identifier, "\"", "\"\"") + "\""
}
