package ztime_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/milagre/zote/go/ztime"
	"github.com/stretchr/testify/assert"
)

func TestNewTime(t *testing.T) {
	now := time.Now()
	timeVal := ztime.NewTime(now, ztime.ResolutionNano)
	assert.Equal(t, now, timeVal.Time)
	assert.Equal(t, ztime.ResolutionNano, timeVal.Resolution())
}

func TestTime_String(t *testing.T) {
	tests := []struct {
		name     string
		time     ztime.Time
		expected string
	}{
		{
			name:     "second resolution",
			time:     ztime.NewTime(time.Date(2024, 3, 20, 15, 4, 5, 0, time.UTC), ztime.ResolutionSecond),
			expected: "15:04:05",
		},
		{
			name:     "millisecond resolution",
			time:     ztime.NewTime(time.Date(2024, 3, 20, 15, 4, 5, 123000000, time.UTC), ztime.ResolutionMilli),
			expected: "15:04:05.123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.time.String())
		})
	}
}

func TestTime_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		time    ztime.Time
		want    string
		wantErr bool
	}{
		{
			name: "second resolution",
			time: ztime.NewTime(time.Date(2024, 3, 20, 15, 4, 5, 0, time.UTC), ztime.ResolutionSecond),
			want: `"15:04:05"`,
		},
		{
			name: "millisecond resolution",
			time: ztime.NewTime(time.Date(2024, 3, 20, 15, 4, 5, 123000000, time.UTC), ztime.ResolutionMilli),
			want: `"15:04:05.123"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.time)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(data))
		})
	}
}

func TestTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    ztime.Time
		wantErr bool
	}{
		{
			name: "second resolution",
			json: `"15:04:05"`,
			want: ztime.NewTime(time.Date(0, 1, 1, 15, 4, 5, 0, time.UTC), ztime.ResolutionSecond),
		},
		{
			name: "null time",
			json: "null",
		},
		{
			name:    "invalid format",
			json:    `"15-04-05"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tm ztime.Time
			err := json.Unmarshal([]byte(tt.json), &tm)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.json != "null" {
				assert.Equal(t, tt.want.String(), tm.String())
			}
		})
	}
}

func TestTime_Scan(t *testing.T) {
	testTime := time.Date(2024, 3, 20, 15, 4, 5, 123456789, time.UTC)
	tests := []struct {
		name    string
		input   interface{}
		want    ztime.Time
		wantErr bool
	}{
		{
			name:  "string input",
			input: "15:04:05",
			want:  ztime.NewTime(time.Date(0, 1, 1, 15, 4, 5, 0, time.UTC), ztime.ResolutionSecond),
		},
		{
			name:  "[]byte input",
			input: []byte("15:04:05"),
			want:  ztime.NewTime(time.Date(0, 1, 1, 15, 4, 5, 0, time.UTC), ztime.ResolutionSecond),
		},
		{
			name:  "time.Time input",
			input: testTime,
			want:  ztime.NewTime(testTime, ztime.ResolutionNano),
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
			name:    "invalid time string",
			input:   "invalid time",
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
			var tm ztime.Time
			err := tm.Scan(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.input != nil {
				assert.Equal(t, tt.want.String(), tm.String())
			}
		})
	}
}

func TestTime_UnmarshalJSON_InvalidJSON(t *testing.T) {
	var tm ztime.Time
	err := tm.UnmarshalJSON([]byte(`{"invalid": "json"}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling ztime.Time: parsing json string")
}

func TestTime_UnmarshalJSON_InvalidTimeFormat(t *testing.T) {
	var tm ztime.Time
	err := tm.UnmarshalJSON([]byte(`"invalid time"`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling ztime.Time: parsing")
}

func TestTime_On(t *testing.T) {
	timeVal := ztime.NewTime(time.Date(0, 1, 1, 15, 4, 5, 123456789, time.UTC), ztime.ResolutionNano)
	date := ztime.NewDate(time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC))
	tz := ztime.NewTimezone(time.UTC)

	result := timeVal.On(date, tz)
	expected := time.Date(2024, 3, 20, 15, 4, 5, 123456789, time.UTC)
	assert.Equal(t, expected, result)
}

func TestTime_Value(t *testing.T) {
	now := time.Now()
	timeVal := ztime.NewTime(now, ztime.ResolutionNano)
	val, err := timeVal.Value()
	assert.NoError(t, err)
	assert.Equal(t, now, val)
}

func TestTime_Resolution(t *testing.T) {
	timeVal := ztime.NewTime(time.Now(), ztime.ResolutionMilli)
	assert.Equal(t, ztime.ResolutionMilli, timeVal.Resolution())
}

func TestTime_SetResolution(t *testing.T) {
	timeVal := ztime.NewTime(time.Now(), ztime.ResolutionSecond)
	assert.Equal(t, ztime.ResolutionSecond, timeVal.Resolution())

	timeVal.SetResolution(ztime.ResolutionNano)
	assert.Equal(t, ztime.ResolutionNano, timeVal.Resolution())
}

func TestTime_UnmarshalJSON_MilliMicroNano(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected string
		res      ztime.Resolution
	}{
		{
			name:     "millisecond resolution",
			json:     `"15:04:05.123"`,
			expected: "15:04:05.123",
			res:      ztime.ResolutionMilli,
		},
		{
			name:     "microsecond resolution",
			json:     `"15:04:05.123456"`,
			expected: "15:04:05.123456",
			res:      ztime.ResolutionMicro,
		},
		{
			name:     "nanosecond resolution",
			json:     `"15:04:05.123456789"`,
			expected: "15:04:05.123456789",
			res:      ztime.ResolutionNano,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tm ztime.Time
			err := json.Unmarshal([]byte(tt.json), &tm)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, tm.String())
			assert.Equal(t, tt.res, tm.Resolution())
		})
	}
}
