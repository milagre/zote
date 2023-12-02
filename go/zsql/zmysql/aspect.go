package zmysql

import (
	"database/sql"
	"fmt"

	"github.com/milagre/zote/go/zcmd"
	"github.com/milagre/zote/go/zcmd/zaspect"
	"github.com/milagre/zote/go/zsql"
)

var _ zcmd.Aspect = Aspect{}

type Aspect struct {
	name string
}

func NewAspect(name string) Aspect {
	return Aspect{
		name: name,
	}
}

func (a Aspect) Apply(c zcmd.Configurable) {
	c.AddString(a.host()).Default("localhost")
	c.AddString(a.user())
	c.AddString(a.pass())
	c.AddString(a.database()).Default("mysql")
	c.AddInt(a.port()).Default(3306)
}

func (a Aspect) Connection(env zcmd.Env, options zsql.Options) (*zsql.Connection, error) {
	dsn := TCPConnectionString(
		env.String(a.user()),
		env.String(a.pass()),
		env.String(a.host()),
		env.Int(a.port()),
		env.String(a.database()),
		options,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening mysql connection: %w", err)
	}

	return zsql.NewConnection(db, driver), nil
}

// Option constructors

func (a Aspect) port() string {
	return zaspect.Prefix(a.name, "mysql-port")
}

func (a Aspect) host() string {
	return zaspect.Prefix(a.name, "mysql-host")
}

func (a Aspect) user() string {
	return zaspect.Prefix(a.name, "mysql-user")
}

func (a Aspect) pass() string {
	return zaspect.Prefix(a.name, "mysql-pass")
}

func (a Aspect) database() string {
	return zaspect.Prefix(a.name, "mysql-database")
}
