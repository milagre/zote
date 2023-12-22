package zlogcmd

import (
	"strings"

	"github.com/milagre/zote/go/zcmd"
	"github.com/milagre/zote/go/zlog"
)

var _ zcmd.Aspect = Aspect{}

type Aspect struct {
	defaultLevel zlog.Level
}

func NewLog(defaultLevel zlog.Level) Aspect {
	return Aspect{
		defaultLevel: defaultLevel,
	}
}

func (a Aspect) Apply(c zcmd.Configurable) {
	c.AddString("log-level").Default("info")
}

func (a Aspect) LogLevel(e zcmd.Env) zlog.Level {
	switch strings.ToLower(e.String("log-level")) {
	default:
		return a.defaultLevel

	case "fatal", "panic", "ftl":
		return zlog.LevelFatal

	case "error", "err":
		return zlog.LevelError

	case "warning", "warn", "wrn":
		return zlog.LevelWarn

	case "info", "inf", "default":
		return zlog.LevelInfo

	case "debug", "dbg":
		return zlog.LevelDebug

	case "trace", "trc":
		return zlog.LevelTrace
	}
}
