package ztime

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

var _ json.Marshaler = Unix{}
var _ json.Unmarshaler = &Unix{}

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

func (t *Unix) SetResolution(res Resolution) {
	t.res = res
}

func (t Unix) String() string {
	var s int64
	switch t.res {
	default:
	case ResolutionSecond:
		s = t.time.Unix()
	case ResolutionMilli:
		s = t.time.UnixMilli()
	case ResolutionMicro:
		s = t.time.UnixMicro()
	case ResolutionNano:
		s = t.time.UnixNano()
	}

	return strconv.FormatInt(s, 10)
}

func (t Unix) MarshalJSON() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t *Unix) UnmarshalJSON(b []byte) error {
	i, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return fmt.Errorf("parsing timestamp as int: %w", err)
	}

	length := len(b)
	switch {
	case length <= 12:
		t.time = time.Unix(i, 0)
		t.res = ResolutionSecond
	case length <= 15:
		t.time = time.UnixMilli(i)
		t.res = ResolutionMilli
	case length <= 18:
		t.time = time.UnixMicro(i)
		t.res = ResolutionMicro
	case length <= 21:
		t.time = time.Unix(0, i)
		t.res = ResolutionNano
	default:
		return fmt.Errorf("parsing identifying timestamp resolution")
	}

	return nil
}
