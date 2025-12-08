package zsqlite3

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/milagre/zote/go/zelement/zmethod"
	"github.com/milagre/zote/go/zsql"
)

var Driver zsql.Driver = driver{}

type driver struct{}

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

func (d driver) EscapeFulltextSearch(search string) string {
	return EscapeString(search)
}

func (d driver) PrepareMethod(m string) *string {
	var result *string

	switch zmethod.Method(m) {
	case zmethod.Contains:
		v := "INSTR(%s, %s) > 0"
		result = &v
	}

	return result
}

func (d driver) IsConflictError(err error) bool {
	var sqliteErr *sqlite.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT
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
	pool, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite3 connection: %w", err)
	}

	pool.SetConnMaxIdleTime(5 * time.Minute)
	pool.SetMaxIdleConns((poolSize / 2) + 1)
	pool.SetMaxOpenConns(poolSize)

	return zsql.NewConnection(pool, Driver), nil
}

func EscapeString(value string) string {
	var sb strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		switch c {
		case '\\', 0, '\n', '\r', '\'', '"', '`':
			sb.WriteByte('\\')
			sb.WriteByte(c)
		case '\032':
			sb.WriteByte('\\')
			sb.WriteByte('Z')
		default:
			sb.WriteByte(c)
		}
	}
	return sb.String()
}
