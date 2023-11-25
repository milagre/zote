package zlog

import (
	"fmt"
	"sync"
)

type logger struct {
	destinations []Destination
	fields       Fields
	level        Level
	lock         sync.Mutex
}

func New(level Level, destinations ...Destination) Logger {
	return new(level, destinations...)
}

func new(level Level, destinations ...Destination) *logger {
	return &logger{
		destinations: append([]Destination{}, destinations...),
		level:        level,
		fields:       Fields{},
	}
}

func (l *logger) AddDestination(d Destination) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.destinations = append(l.destinations, d)
}

func (l *logger) Level() Level         { return l.level }
func (l *logger) SetLevel(level Level) { l.level = level }

func (l *logger) Fatal(message string)                      { l.send(LevelFatal, message) }
func (l *logger) Fatalf(format string, args ...interface{}) { l.sendf(LevelFatal, format, args...) }
func (l *logger) Error(message string)                      { l.send(LevelError, message) }
func (l *logger) Errorf(format string, args ...interface{}) { l.sendf(LevelError, format, args...) }
func (l *logger) Warn(message string)                       { l.send(LevelWarn, message) }
func (l *logger) Warnf(format string, args ...interface{})  { l.sendf(LevelWarn, format, args...) }
func (l *logger) Info(message string)                       { l.send(LevelInfo, message) }
func (l *logger) Infof(format string, args ...interface{})  { l.sendf(LevelInfo, format, args...) }
func (l *logger) Debug(message string)                      { l.send(LevelDebug, message) }
func (l *logger) Debugf(format string, args ...interface{}) { l.sendf(LevelDebug, format, args...) }
func (l *logger) Trace(message string)                      { l.send(LevelTrace, message) }
func (l *logger) Tracef(format string, args ...interface{}) { l.sendf(LevelTrace, format, args...) }

func (l *logger) WithField(key string, value interface{}) Logger { return l.withField(key, value) }
func (l *logger) WithFields(fields Fields) Logger                { return l.withFields(fields) }

func (l *logger) send(level Level, message string) {
	for _, d := range l.destinations {
		d.Send(level, l.fields, message)
	}
}

func (l *logger) sendf(level Level, format string, args ...interface{}) {
	l.send(level, fmt.Sprintf(format, args...))
}

func (l *logger) withFields(fields Fields) Logger {
	result := new(l.level, l.destinations...)

	for k, v := range l.fields {
		result.fields[k] = v
	}

	for k, v := range fields {
		result.fields[k] = v
	}

	return result
}

func (l *logger) withField(key string, value interface{}) Logger {
	return l.withFields(Fields{
		key: value,
	})
}
