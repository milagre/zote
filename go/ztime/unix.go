package ztime

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

var _ json.Marshaler = Unix{}
var _ json.Unmarshaler = &Unix{}

type Resolution time.Duration

const (
	ResolutionSecond = Resolution(time.Second)
	ResolutionMilli  = Resolution(time.Millisecond)
	ResolutionMicro  = Resolution(time.Microsecond)
	ResolutionNano   = Resolution(time.Nanosecond)
)

type Unix struct {
	time time.Time
	res  Resolution
}

func NewUnix(t time.Time, res Resolution) Unix {
	return Unix{
		time: t,
		res:  res,
	}
}

func (t Unix) Time() time.Time {
	return t.time
}

func (t Unix) Resolution() Resolution {
	return t.res
}

func (t Unix) MarshalJSON() ([]byte, error) {
	var s int64

	switch t.res {
	case ResolutionSecond:
		s = t.time.Unix()
	case ResolutionMilli:
		s = t.time.UnixMilli()
	case ResolutionMicro:
		s = t.time.UnixMicro()
	case ResolutionNano:
		s = t.time.UnixNano()
	default:
		return nil, fmt.Errorf("invalid timestamp resolution")
	}

	return []byte(strconv.FormatInt(s, 10)), nil
}

func (t *Unix) UnmarshalJSON(b []byte) error {
	i, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return fmt.Errorf("parsing timestamp as int: %w", err)
	}

	switch len(b) {
	case 10, 11, 12:
		t.time = time.Unix(i, 0)
		t.res = ResolutionSecond
	case 13, 14, 15:
		t.time = time.UnixMilli(i)
		t.res = ResolutionMilli
	case 16, 17, 18:
		t.time = time.UnixMicro(i)
		t.res = ResolutionMicro
	case 19, 20, 21:
		t.time = time.Unix(0, i)
		t.res = ResolutionNano
	default:
		return fmt.Errorf("parsing identifying timestamp resolution")
	}

	return nil
}
