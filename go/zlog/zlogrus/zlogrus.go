package zlogrus

import (
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"github.com/milagre/zote/go/zlog"
)

func New(level zlog.Level) zlog.Destination {
	l := logrus.New()
	l.Formatter = &prefixed.TextFormatter{
		//DisableColors:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
		ForceFormatting: true,
	}
	return Wrap(l, level)
}

func Wrap(l *logrus.Logger, level zlog.Level) zlog.Destination {
	l.Level = levels[level]
	return &destination{
		logger: l,
		level:  level,
		fields: logrus.Fields{},
	}
}

var levels map[zlog.Level]logrus.Level

func init() {
	levels = map[zlog.Level]logrus.Level{
		zlog.LevelFatal: logrus.FatalLevel,
		zlog.LevelError: logrus.ErrorLevel,
		zlog.LevelWarn:  logrus.WarnLevel,
		zlog.LevelInfo:  logrus.InfoLevel,
		zlog.LevelDebug: logrus.DebugLevel,
		zlog.LevelTrace: logrus.TraceLevel,
	}
}

type destination struct {
	logger *logrus.Logger
	level  zlog.Level
	fields logrus.Fields
}

func (d *destination) Send(level zlog.Level, fields zlog.Fields, message string) {
	if level > d.level {
		return
	}

	d.logger.
		WithFields((map[string]interface{})(fields)).
		Log(levels[level], message)
}

func (d *destination) Level() zlog.Level {
	return d.level
}

func (d *destination) SetLevel(level zlog.Level) {
	d.level = level
	d.logger.Level = levels[level]
}
