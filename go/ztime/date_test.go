package ztime_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/milagre/zote/go/ztime"
	"github.com/stretchr/testify/assert"
)

func TestNewDate(t *testing.T) {
	now := time.Now()
	date := ztime.NewDate(now)
	assert.Equal(t, now, date.Time)
}

func TestDate_String(t *testing.T) {
	date := ztime.NewDate(time.Date(2024, 3, 20, 15, 4, 5, 0, time.UTC))
	assert.Equal(t, "2024-03-20", date.String())
}

func TestDate_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		date    ztime.Date
		want    string
		wantErr bool
	}{
		{
			name: "valid date",
			date: ztime.NewDate(time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)),
			want: `"2024-03-20"`,
		},
		{
			name: "zero date",
			date: ztime.Date{},
			want: `"0001-01-01"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.date)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(data))
		})
	}
}

func TestDate_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    ztime.Date
		wantErr bool
	}{
		{
			name: "valid date",
			json: `"2024-03-20"`,
			want: ztime.NewDate(time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)),
		},
		{
			name: "null date",
			json: "null",
		},
		{
			name:    "invalid format",
			json:    `"2024/03/20"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d ztime.Date
			err := json.Unmarshal([]byte(tt.json), &d)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.json != "null" {
				assert.Equal(t, tt.want.Format(time.DateOnly), d.Format(time.DateOnly))
			}
		})
	}
}

func TestDate_Scan(t *testing.T) {
	date := ztime.NewDate(time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC))
	tests := []struct {
		name    string
		input   interface{}
		want    ztime.Date
		wantErr bool
	}{
		{
			name:  "string input",
			input: "2024-03-20",
			want:  date,
		},
		{
			name:  "time.Time input",
			input: date.Time,
			want:  date,
		},
		{
			name:    "int64 input",
			input:   int64(123),
			wantErr: true,
		},
		{
			name:    "float64 input",
			input:   float64(123.45),
			wantErr: true,
		},
		{
			name:    "bool input",
			input:   true,
			wantErr: true,
		},
		{
			name:  "nil input",
			input: nil,
		},
		{
			name:    "invalid date string",
			input:   "invalid date",
			wantErr: true,
		},
		{
			name:    "unhandled type",
			input:   struct{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d ztime.Date
			err := d.Scan(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.input != nil {
				assert.Equal(t, tt.want.Format(time.DateOnly), d.Format(time.DateOnly))
			}
		})
	}
}

func TestDate_UnmarshalJSON_InvalidJSON(t *testing.T) {
	var d ztime.Date
	err := d.UnmarshalJSON([]byte(`{"invalid": "json"}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling ztime.Date: parsing json string")
}

func TestDate_UnmarshalJSON_InvalidDateFormat(t *testing.T) {
	var d ztime.Date
	err := d.UnmarshalJSON([]byte(`"invalid date"`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling ztime.Date: parsing")
}

func TestDate_At(t *testing.T) {
	date := ztime.NewDate(time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC))
	timeVal := ztime.NewTime(time.Date(0, 1, 1, 15, 4, 5, 123456789, time.UTC), ztime.ResolutionNano)
	tz := ztime.NewTimezone(time.UTC)

	result := date.At(timeVal, tz)
	expected := time.Date(2024, 3, 20, 15, 4, 5, 123456789, time.UTC)
	assert.Equal(t, expected, result)
}

func TestDate_Value(t *testing.T) {
	now := time.Now()
	date := ztime.NewDate(now)
	val, err := date.Value()
	assert.NoError(t, err)
	assert.Equal(t, now, val)
}

func TestDate_UnmarshalJSON_ByteSlice(t *testing.T) {
	var d ztime.Date
	err := d.Scan([]byte("2024-03-20"))
	assert.NoError(t, err)
	expected := ztime.NewDate(time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, expected.Format(time.DateOnly), d.Format(time.DateOnly))
}
