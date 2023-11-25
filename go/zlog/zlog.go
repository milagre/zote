package zlog

type Fields map[string]interface{}

type Level int

const (
	_ Level = iota
	LevelFatal
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
	LevelTrace
)

type Logger interface {
	AddDestination(d Destination)

	Level() Level
	SetLevel(l Level)

	Fatal(message string)
	Fatalf(format string, args ...interface{})

	Error(message string)
	Errorf(format string, args ...interface{})

	Warn(message string)
	Warnf(format string, args ...interface{})

	Info(message string)
	Infof(format string, args ...interface{})

	Debug(message string)
	Debugf(format string, args ...interface{})

	Trace(message string)
	Tracef(format string, args ...interface{})

	WithField(key string, value interface{}) Logger
	WithFields(fields Fields) Logger
}

type Destination interface {
	Send(level Level, fields Fields, message string)
	Level() Level
	SetLevel(l Level)
}
