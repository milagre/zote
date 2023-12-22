package zamqpcmd

import (
	"fmt"

	"github.com/milagre/zote/go/zamqp"
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
	c.AddString(a.host()).Default("localhost")
	c.AddString(a.user())
	c.AddString(a.pass())
	c.AddString(a.vhost()).Default("/")
	c.AddInt(a.port()).Default(3306)
}

func (a Aspect) Details(env zcmd.Env) zamqp.ConnectionDetails {
	return zamqp.NewConnectionDetails(
		env.String(a.user()),
		env.String(a.pass()),
		env.String(a.host()),
		env.Int(a.port()),
		env.String(a.vhost()),
	)
}

func (a Aspect) Connection(env zcmd.Env) (zamqp.Connection, error) {
	conn, err := zamqp.Dial(a.Details(env))
	if err != nil {
		return zamqp.Connection{}, fmt.Errorf("opening amqp connection: %w", err)
	}

	return conn, nil
}

func (a Aspect) ConnectionProvider(env zcmd.Env) func() (zamqp.Connection, error) {
	return func() (zamqp.Connection, error) {
		return a.Connection(env)
	}
}

// Option constructors

func (a Aspect) port() string {
	return zaspect.Format("amqp-%s-port", a.name)
}

func (a Aspect) host() string {
	return zaspect.Format("amqp-%s-host", a.name)
}

func (a Aspect) user() string {
	return zaspect.Format("amqp-%s-user", a.name)
}

func (a Aspect) pass() string {
	return zaspect.Format("amqp-%s-pass", a.name)
}

func (a Aspect) vhost() string {
	return zaspect.Format("amqp-%s-vhost", a.name)
}
