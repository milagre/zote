package zredis

import (
	"fmt"

	"github.com/redis/go-redis/v9"

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
	c.AddInt(a.port()).Default(6379)
}

func (a Aspect) Client(env zcmd.Env) *redis.Client {
	return redis.NewClient(
		&redis.Options{
			Addr: fmt.Sprintf(
				"%s:%d",
				env.String(a.host()),
				env.Int(a.port()),
			),
		},
	)
}

// Option constructors

func (a Aspect) port() string {
	return zaspect.Format("redis-%s-port", a.name)
}

func (a Aspect) host() string {
	return zaspect.Format("redis-%s-host", a.name)
}
