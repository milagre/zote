package ztime

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

var (
	_ json.Marshaler   = Month{}
	_ json.Unmarshaler = &Month{}
	_ sql.Scanner      = &Month{}
	_ driver.Valuer    = Month{}
)

type Month struct {
	time.Time
}

const MonthFormat = "2006-01"

func NewMonth(t time.Time) Month {
	return Month{
		Time: t,
	}
}

func NewMonthFromValues(year int, month time.Month) Month {
	return NewMonth(time.Date(year, month, 1, 0, 0, 0, 0, time.UTC))
}

func NewMonthFromDate(d Date) Month {
	return NewMonthFromValues(d.Year(), d.Month())
}

func (m Month) String() string {
	return m.Format(MonthFormat)
}

func (m *Month) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		m = nil
		return nil
	}

	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return fmt.Errorf("unmarshalling ztime.Month: parsing json string: %w", err)
	}

	val, err := ParseMonth(s)
	if err != nil {
		return fmt.Errorf("unmarshalling ztime.Month: parsing: %w", err)
	}

	*m = val

	return nil
}

func (m Month) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (m *Month) Scan(data any) error {
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
		return fmt.Errorf("scanning ztime.Month: invalid input type: %+T", data)

	case string:
		val, err := time.Parse(MonthFormat, string(input))
		if err != nil {
			return fmt.Errorf("scanning ztime.Month: parsing date: %w", err)
		}
		*m = Month{val}
		return nil

	case time.Time:
		*m = Month{input}
		return nil

	case nil:
		return nil
	}

	return fmt.Errorf("scanning ztime.Month: unrecognized type: %+T", data)
}

func (m Month) Value() (driver.Value, error) {
	return m.Time, nil
}

func (m Month) Prev() Month {
	return NewMonth(m.Time.AddDate(0, -1, 0))
}

func (m Month) Next() Month {
	return NewMonth(m.Time.AddDate(0, 1, 0))
}

func (m Month) Add(n int) Month {
	return NewMonth(m.Time.AddDate(0, n, 0))
}

func (m Month) OnDay(day int) Date {
	return NewDateFromValues(m.Year(), m.Month(), day)
}

// Ptr returns a pointer to a copy of the Month. Useful when you simply need a pointer value instead of a value after construction.
func (m Month) Ptr() *Month {
	return &m
}

func ParseMonth(s string) (Month, error) {
	val, err := time.Parse(MonthFormat, s)
	if err != nil {
		return Month{}, fmt.Errorf("unmarshalling ztime.Month: parsing date: %w", err)
	}

	return Month{val}, nil
}
