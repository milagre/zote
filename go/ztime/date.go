package ztime

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

var (
	_ json.Marshaler   = Date{}
	_ json.Unmarshaler = &Date{}
	_ sql.Scanner      = &Date{}
	_ driver.Valuer    = Date{}
)

type Date struct {
	time.Time
}

func NewDate(t time.Time) Date {
	return Date{
		Time: t,
	}
}

func NewDateFromValues(year int, month time.Month, day int) Date {
	return NewDate(time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
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
		return fmt.Errorf("unmarshalling ztime.Date: parsing json string: %w", err)
	}

	val, err := parseDate(s)
	if err != nil {
		return fmt.Errorf("unmarshalling ztime.Date: parsing: %w", err)
	}

	*d = val

	return nil
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Date) Scan(data any) error {
	target := data

	// Convert []uint8/[]byte into string explicitly, it weirdly doesnt match in
	// the type switch below on mysql Time columns
	if b, ok := data.([]byte); ok {
		target = string(b)
	}

	switch input := target.(type) {
	case int64:
	case float64:
	case bool:
		return fmt.Errorf("scanning ztime.Date: invalid input type: %+T", data)

	case string:
		val, err := time.Parse(time.DateOnly, string(input))
		if err != nil {
			return fmt.Errorf("scanning ztime.Date: parsing date: %w", err)
		}
		*d = Date{val}
		return nil

	case time.Time:
		*d = Date{input}
		return nil

	case nil:
		return nil
	}

	return fmt.Errorf("scanning ztime.Date: unrecognized type: %+T", data)
}

func (d Date) Value() (driver.Value, error) {
	return d.Time, nil
}

func (d Date) At(t Time, tz Timezone) time.Time {
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
		return Date{}, fmt.Errorf("unmarshalling ztime.Date: parsing date: %w", err)
	}

	return Date{val}, nil
}
