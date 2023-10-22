package mysql

import (
	"database/sql"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/spf13/cast"

	"github.com/milagre/zote/go/cmd"
	"github.com/milagre/zote/go/cmd/aspects"
	zotesql "github.com/milagre/zote/go/sql"
)

var _ cmd.Aspect = Aspect{}

type Aspect struct {
	name string
}

func NewAspect(name string) Aspect {
	return Aspect{
		name: name,
	}
}

func (a Aspect) Apply(c cmd.Configurable) {
	c.AddString(a.host()).Default("localhost")
	c.AddString(a.user())
	c.AddString(a.pass())
	c.AddString(a.database()).Default("mysql")
	c.AddInt(a.port()).Default(3306)
}

func (a Aspect) Connection(env cmd.Env, options zotesql.Options) (*zotesql.Connection, error) {
	cfg := mysql.NewConfig()
	cfg.User = env.String(a.user())
	cfg.Passwd = env.String(a.pass())
	cfg.DBName = env.String(a.database())
	cfg.Addr = fmt.Sprintf("%s:%d", env.String(a.host()), env.Int(a.port()))
	cfg.Params = cast.ToStringMapString(options)

	dsn := cfg.FormatDSN()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening mysql connection: %w", err)
	}

	return zotesql.NewConnection(db, driver), nil
}

// Option constructors

func (a Aspect) port() string {
	return aspects.Prefix(a.name, "mysql-port")
}

func (a Aspect) host() string {
	return aspects.Prefix(a.name, "mysql-host")
}

func (a Aspect) user() string {
	return aspects.Prefix(a.name, "mysql-user")
}

func (a Aspect) pass() string {
	return aspects.Prefix(a.name, "mysql-pass")
}

func (a Aspect) database() string {
	return aspects.Prefix(a.name, "mysql-database")
}
