package zlogcmd

import (
	"fmt"
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

func (a Aspect) LogLevel(e zcmd.Env) (zlog.Level, error) {
	switch strings.ToLower(e.String("log-level")) {
	case "":
		return a.defaultLevel, nil

	case "fatal", "panic", "ftl":
		return zlog.LevelFatal, nil

	case "error", "err":
		return zlog.LevelError, nil

	case "warning", "warn", "wrn":
		return zlog.LevelWarn, nil

	case "info", "inf", "default":
		return zlog.LevelInfo, nil

	case "debug", "dbg":
		return zlog.LevelDebug, nil

	case "trace", "trc":
		return zlog.LevelTrace, nil
	}

	return zlog.LevelInfo, fmt.Errorf("unknown log level name provided")
}
