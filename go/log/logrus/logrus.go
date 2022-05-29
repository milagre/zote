package logrus

import (
	"github.com/sirupsen/logrus"

	zotelog "github.com/milagre/zote/go/log"
)

func New(level zotelog.Level) zotelog.Destination {
	return Wrap(logrus.New(), level)
}

func Wrap(l *logrus.Logger, level zotelog.Level) zotelog.Destination {
	l.Level = levels[level]
	return &destination{
		logger: l,
		level:  level,
		fields: logrus.Fields{},
	}
}

var levels map[zotelog.Level]logrus.Level

func init() {
	levels = map[zotelog.Level]logrus.Level{
		zotelog.LevelFatal: logrus.FatalLevel,
		zotelog.LevelError: logrus.ErrorLevel,
		zotelog.LevelWarn:  logrus.WarnLevel,
		zotelog.LevelInfo:  logrus.InfoLevel,
		zotelog.LevelDebug: logrus.DebugLevel,
		zotelog.LevelTrace: logrus.TraceLevel,
	}
}

type destination struct {
	logger *logrus.Logger
	level  zotelog.Level
	fields logrus.Fields
}

func (d *destination) Send(level zotelog.Level, fields zotelog.Fields, message string) {
	if level > d.level {
		return
	}

	d.logger.
		WithFields((map[string]interface{})(fields)).
		Log(levels[level], message)
}

func (d *destination) Level() zotelog.Level {
	return d.level
}

func (d *destination) SetLevel(level zotelog.Level) {
	d.level = level
	d.logger.Level = levels[level]
}
