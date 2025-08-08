package ztimescaledb

import (
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/lib/pq"

	"github.com/milagre/zote/go/zcmd"
	"github.com/milagre/zote/go/zcmd/zaspect"
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
	c.AddString(a.scheme()).Default("postgres")
	c.AddString(a.host()).Default("localhost")
	c.AddInt(a.port()).Default(5432)
	c.AddString(a.database())
	c.AddString(a.user())
	c.AddString(a.password())
	c.AddString(a.sslmode()).Default("disable")
}

type Client struct {
	*sql.DB

	database string
}

func (c Client) DefaultDatabase() string {
	return c.database
}

func (a Aspect) Client(env zcmd.Env) Client {
	uri := url.URL{
		Scheme: env.String(a.scheme()),
		Host:   fmt.Sprintf("%s:%d", env.String(a.host()), env.Int(a.port())),
		User:   url.UserPassword(env.String(a.user()), env.String(a.password())),
		Path:   "/" + env.String(a.database()),
	}

	// Add SSL mode as query parameter
	q := uri.Query()
	q.Set("sslmode", env.String(a.sslmode()))
	uri.RawQuery = q.Encode()

	db, err := sql.Open("postgres", uri.String())
	if err != nil {
		panic(fmt.Sprintf("failed to create TimescaleDB client: %v", err))
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		panic(fmt.Sprintf("failed to connect to TimescaleDB: %v", err))
	}

	return Client{
		DB:       db,
		database: env.String(a.database()),
	}
}

// Option constructors

func (a Aspect) scheme() string {
	return a.opt("scheme")
}

func (a Aspect) host() string {
	return a.opt("host")
}

func (a Aspect) port() string {
	return a.opt("port")
}

func (a Aspect) database() string {
	return a.opt("database")
}

func (a Aspect) user() string {
	return a.opt("user")
}

func (a Aspect) password() string {
	return a.opt("password")
}

func (a Aspect) sslmode() string {
	return a.opt("sslmode")
}

func (a Aspect) opt(opt string) string {
	return zaspect.Format("timescaledb-%s-%s", a.name, opt)
}
