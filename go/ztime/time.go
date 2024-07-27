package ztime

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

var _ json.Marshaler = Time{}
var _ json.Unmarshaler = &Time{}
var _ sql.Scanner = &Time{}
var _ driver.Valuer = &Time{}

var timeFormats map[Resolution]string

func init() {
	timeFormats = map[Resolution]string{
		ResolutionSecond: "15:04:05",
		ResolutionMilli:  "15:04:05.999",
		ResolutionMicro:  "15:04:05.999999",
		ResolutionNano:   "15:04:05.999999999",
	}
}

type Time struct {
	time.Time
	res Resolution
}

func NewTime(t time.Time, res Resolution) Time {
	return Time{
		Time: t,
		res:  res,
	}
}

func (t *Time) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		t = nil
		return nil
	}

	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return fmt.Errorf("unmarshalling time: parsing json string: %w", err)
	}

	val, err := parseTime(s)
	if err != nil {
		return fmt.Errorf("unmarshalling time: parsing: %w", err)
	}

	*t = val

	return nil
}

func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Format(timeFormats[t.res]))
}

func (t *Time) Scan(data any) error {
	switch input := data.(type) {
	case int64:
	case float64:
	case bool:
		return fmt.Errorf("scanning time: invalid input type: %+T", data)

	case []byte:
	case string:
		val, err := parseTime(string(input))
		if err != nil {
			return fmt.Errorf("scanning time: parsing: %w", err)
		}
		*t = val
		return nil

	case time.Time:
		*t = Time{Time: input, res: ResolutionNano}
		return nil

	case nil:
		return nil
	}

	return fmt.Errorf("scanning time: unrecognized type: %+T", data)
}

func (t Time) Value() (driver.Value, error) {
	return t.Time, nil
}

func (t *Time) On(d Date, tz Timezone) time.Time {
	return time.Date(
		d.Year(),
		d.Month(),
		d.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
		t.Nanosecond(),
		tz.Location,
	)
}

func parseTime(s string) (Time, error) {
	var res Resolution
	for r, f := range timeFormats {
		if len(s) == len(f) {
			res = r
			break
		}
	}

	val, err := time.Parse(timeFormats[res], s)
	if err != nil {
		return Time{}, fmt.Errorf("parsing date: %w", err)
	}

	return Time{Time: val, res: res}, nil
}
