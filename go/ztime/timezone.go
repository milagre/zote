package ztime

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

var _ json.Marshaler = Timezone{}
var _ json.Unmarshaler = &Timezone{}
var _ sql.Scanner = &Timezone{}
var _ driver.Valuer = &Timezone{}

type Timezone struct {
	*time.Location
}

func NewTimezone(l *time.Location) Timezone {
	return Timezone{
		Location: l,
	}
}

func (tz *Timezone) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		tz = nil
		return nil
	}

	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return fmt.Errorf("unmarshalling timezone: parsing value into string: %w", err)
	}

	l, err := time.LoadLocation(s)
	if err != nil {
		return fmt.Errorf("unmarshalling timezone: loading: %w", err)
	}

	*tz = Timezone{l}

	return nil
}

func (tz Timezone) MarshalJSON() ([]byte, error) {
	return json.Marshal(tz.String())
}

func (tz *Timezone) Scan(data any) error {
	switch input := data.(type) {
	case int64:
	case float64:
	case bool:
	case time.Time:
		return fmt.Errorf("scanning timezone: invalid input type: %+T", data)

	case []byte:
	case string:
		val, err := time.LoadLocation(string(input))
		if err != nil {
			return fmt.Errorf("scanning timezone: loading location: %w", err)
		}
		*tz = Timezone{val}
		return nil

	case nil:
		return nil
	}

	return fmt.Errorf("scanning timezone: unrecognized type: %+T", data)
}

func (tz Timezone) Value() (driver.Value, error) {
	return tz.Location.String(), nil
}
