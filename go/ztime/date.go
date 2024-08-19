package ztime

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

var _ json.Marshaler = Date{}
var _ json.Unmarshaler = &Date{}
var _ sql.Scanner = &Date{}
var _ driver.Valuer = Date{}

type Date struct {
	time.Time
}

func NewDate(t time.Time) Date {
	return Date{
		Time: t,
	}
}

func (d Date) String() string {
	return d.Format(time.DateOnly)
}

func (d *Date) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		d = nil
		return nil
	}

	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return fmt.Errorf("unmarshalling date: parsing json string: %w", err)
	}

	val, err := parseDate(s)
	if err != nil {
		return fmt.Errorf("unmarshalling date: parsing: %w", err)
	}

	*d = val

	return nil
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Date) Scan(data any) error {
	switch input := data.(type) {
	case int64:
	case float64:
	case bool:
		return fmt.Errorf("scanning date: invalid input type: %+T", data)

	case []byte:
	case string:
		val, err := time.Parse(time.DateOnly, string(input))
		if err != nil {
			return fmt.Errorf("scanning date: parsing date: %w", err)
		}
		*d = Date{val}
		return nil

	case time.Time:
		*d = Date{input}
		return nil

	case nil:
		return nil

	}

	return fmt.Errorf("scanning date: unrecognized type: %+T", data)
}

func (d Date) Value() (driver.Value, error) {
	return d.Time, nil
}

func (d *Date) At(t Time, tz Timezone) time.Time {
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

func parseDate(s string) (Date, error) {
	val, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return Date{}, fmt.Errorf("unmarshalling date: parsing date: %w", err)
	}

	return Date{val}, nil
}
