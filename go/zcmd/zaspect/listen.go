package zaspect

import (
	"fmt"

	"github.com/milagre/zote/go/zcmd"
)

var _ zcmd.Aspect = Listen{}

type Listen struct {
	prefix string
}

func NewListenAspect(prefix string) Listen {
	return Listen{
		prefix: prefix,
	}
}

func (a Listen) Apply(c zcmd.Configurable) {
	c.AddString(Prefix(a.prefix, "listen-ip")).Default("0.0.0.0")
	c.AddInt(Prefix(a.prefix, "listen-port")).Default(5000)
}

func (a Listen) ListenPort(env zcmd.Env) int {
	return env.Int(Prefix(a.prefix, "listen-port"))
}

func (a Listen) ListenIP(env zcmd.Env) string {
	return env.String(Prefix(a.prefix, "listen-ip"))
}

func (a Listen) ListenAddr(env zcmd.Env) string {
	return fmt.Sprintf("%s:%d", a.ListenIP(env), a.ListenPort(env))
}
