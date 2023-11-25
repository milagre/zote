package zwarn

import "fmt"

type Warning interface {
	Warning() string
}

type Wrapped interface{}

func Warnf(format string, args ...interface{}) Warning {
	msg := fmt.Sprintf(format, args...)
	w := &warningMultiple{
		msg: msg,
	}
	for _, arg := range args {
		if wrn, ok := arg.(Warning); ok {
			w.wrns = append(w.wrns, wrn)
		} else if err, ok := arg.(error); ok {
			w.errs = append(w.errs, err)
		}
	}

	if len(w.wrns) <= 1 && len(w.errs) <= 1 {
		return &warning{
			msg: msg,
			wrn: append(w.wrns, nil)[0],
			err: append(w.errs, nil)[0],
		}
	}

	return w
}

type warning struct {
	msg string
	wrn Warning
	err error
}

func (e *warning) Warning() string {
	return e.msg
}

func (e *warning) UnwrapError() error {
	return e.err
}

func (e *warning) UnwrapWarning() Warning {
	return e.wrn
}

// Stringer
func (e *warning) String() string {
	return e.Warning()
}

// error
func (e *warning) Error() string {
	return e.Warning()
}

type warningMultiple struct {
	msg  string
	wrns []Warning
	errs []error
}

func (e *warningMultiple) Warning() string {
	return e.msg
}

func (e *warningMultiple) UnwrapErrors() []error {
	return e.errs
}

func (e *warningMultiple) UnwrapWarnings() []Warning {
	return e.wrns
}

// Stringer
func (e *warningMultiple) String() string {
	return e.Warning()
}

// error
func (e *warningMultiple) Error() string {
	return e.Warning()
}
