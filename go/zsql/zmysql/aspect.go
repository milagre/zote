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
	c.AddBool(a.debug())
}

func (a Aspect) Connection(env zcmd.Env, options zsql.Options) (zsql.Connection, error) {
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

	conn := zsql.NewConnection(db, Driver)

	if env.Bool(a.debug()) {
		conn = zsql.NewLoggingConnection(conn)
	}
	return conn, nil
}

// Option constructors

func (a Aspect) port() string {
	return zaspect.Format("mysql-%s-port", a.name)
}

func (a Aspect) host() string {
	return zaspect.Format("mysql-%s-host", a.name)
}

func (a Aspect) user() string {
	return zaspect.Format("mysql-%s-user", a.name)
}

func (a Aspect) pass() string {
	return zaspect.Format("mysql-%s-pass", a.name)
}

func (a Aspect) database() string {
	return zaspect.Format("mysql-%s-database", a.name)
}

func (a Aspect) debug() string {
	return zaspect.Format("mysql-%s-debug", a.name)
}
