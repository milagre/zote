package logcmd

import (
	"fmt"
	"strings"

	"github.com/milagre/zote/go/cmd"
	"github.com/milagre/zote/go/log"
)

var _ cmd.Aspect = Aspect{}

type Aspect struct {
	defaultLevel log.Level
}

func NewLog(defaultLevel log.Level) Aspect {
	return Aspect{
		defaultLevel: defaultLevel,
	}
}

func (a Aspect) Apply(c cmd.Configurable) {
	c.AddString("log-level")
}

func (a Aspect) LogLevel(e cmd.Env) (log.Level, error) {
	switch strings.ToLower(e.String("log-level")) {
	case "":
		return a.defaultLevel, nil

	case "fatal", "panic", "ftl":
		return log.LevelFatal, nil

	case "error", "err":
		return log.LevelError, nil

	case "warning", "warn", "wrn":
		return log.LevelWarn, nil

	case "info", "inf", "default":
		return log.LevelInfo, nil

	case "debug", "dbg":
		return log.LevelDebug, nil

	case "trace", "trc":
		return log.LevelTrace, nil
	}

	return log.LevelInfo, fmt.Errorf("unknown log level name provided")
}
