package aspects

import (
	"fmt"

	"github.com/milagre/zote/go/cmd"
)

var _ cmd.Aspect = Listen{}

type Listen struct {
	prefix string
}

func NewListenAspect(prefix string) Listen {
	return Listen{
		prefix: prefix,
	}
}

func (a Listen) Apply(c cmd.Configurable) {
	c.AddString(Prefix(a.prefix, "listen-ip")).Default("0.0.0.0")
	c.AddInt(Prefix(a.prefix, "listen-port")).Default(5000)
}

func (a Listen) ListenPort(env cmd.Env) int {
	return env.Int(Prefix(a.prefix, "listen-port"))
}

func (a Listen) ListenIP(env cmd.Env) string {
	return env.String(Prefix(a.prefix, "listen-ip"))
}

func (a Listen) ListenAddr(env cmd.Env) string {
	return fmt.Sprintf("%s:%d", a.ListenIP(env), a.ListenPort(env))
}
