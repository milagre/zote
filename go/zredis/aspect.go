package zredis

import (
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/milagre/zote/go/zcmd"
	"github.com/milagre/zote/go/zcmd/zaspect"
)

var _ zcmd.Aspect = ClusterAspect{}

type ClusterAspect struct {
	name string
}

func NewClusterAspect(name string) ClusterAspect {
	return ClusterAspect{
		name: name,
	}
}

func (a ClusterAspect) Apply(c zcmd.Configurable) {
	c.AddString(a.host()).Default("localhost")
	c.AddInt(a.port()).Default(6379)
}

func (a ClusterAspect) Client(env zcmd.Env) *redis.ClusterClient {
	return redis.NewClusterClient(
		&redis.ClusterOptions{
			Addrs: []string{
				fmt.Sprintf("%s:%d", env.String(a.host()), env.Int(a.port())),
			},
		},
	)
}

// Option constructors

func (a ClusterAspect) port() string {
	return zaspect.Format("redis-%s-port", a.name)
}

func (a ClusterAspect) host() string {
	return zaspect.Format("redis-%s-host", a.name)
}
