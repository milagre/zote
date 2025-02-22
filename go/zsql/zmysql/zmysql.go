package zmysql

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/milagre/zote/go/zelement/zmethod"
	"github.com/milagre/zote/go/zsql"
)

var Driver zsql.Driver = driver{}

type driver struct{}

func (d driver) Name() string {
	return "mysql"
}

func (d driver) EscapeTable(t string) string {
	return "`" + strings.ReplaceAll(t, "`", "\\`") + "`"
}

func (d driver) EscapeColumn(c string) string {
	return "`" + strings.ReplaceAll(c, "`", "\\`") + "`"
}

func (d driver) EscapeTableColumn(t string, c string) string {
	return d.EscapeTable(t) + "." + d.EscapeColumn(c)
}

func (d driver) NullSafeEqualityOperator() string {
	return "<=>"
}

func (d driver) EscapeFulltextSearch(search string) string {
	return `"` + EscapeString(search) + `"`
}

func (d driver) PrepareMethod(m string) *string {
	var result *string

	switch zmethod.Method(m) {
	case zmethod.Match:
		v := "MATCH(%s) AGAINST(? IN BOOLEAN MODE)"
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
		"collation":       "utf8mb4_bin",
		"charset":         "utf8mb4",
		"clientFoundRows": true,
		"parseTime":       true,
		"timeout":         5 * time.Second, // Connection timeout only
	}
}

func TCPConnectionString(user string, pass string, host string, port int, db string, opts zsql.Options) string {
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

func Open(dsn string, poolSize int) (zsql.Connection, error) {
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening mysql connection: %w", err)
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
