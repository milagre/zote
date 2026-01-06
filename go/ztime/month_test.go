package ztime_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/milagre/zote/go/ztime"
)

func TestNewMonth(t *testing.T) {
	now := time.Now()
	month := ztime.NewMonth(now)
	assert.Equal(t, now, month.Time)
}

func TestNewMonthFromValues(t *testing.T) {
	month := ztime.NewMonthFromValues(2024, time.March)
	assert.Equal(t, 2024, month.Year())
	assert.Equal(t, time.March, month.Month())
	assert.Equal(t, 1, month.Day())
}

func TestNewMonthFromDate(t *testing.T) {
	date := ztime.NewDateFromValues(2024, time.March, 15)
	month := ztime.NewMonthFromDate(date)
	assert.Equal(t, 2024, month.Year())
	assert.Equal(t, time.March, month.Month())
	assert.Equal(t, 1, month.Day())
}

func TestMonth_String(t *testing.T) {
	month := ztime.NewMonth(time.Date(2024, 3, 20, 15, 4, 5, 0, time.UTC))
	assert.Equal(t, "2024-03", month.String())
}

func TestMonth_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		month   ztime.Month
		want    string
		wantErr bool
	}{
		{
			name:  "valid month",
			month: ztime.NewMonth(time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)),
			want:  `"2024-03"`,
		},
		{
			name:  "zero month",
			month: ztime.Month{},
			want:  `"0001-01"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.month)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(data))
		})
	}
}

func TestMonth_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    ztime.Month
		wantErr bool
	}{
		{
			name: "valid month",
			json: `"2024-03"`,
			want: ztime.NewMonth(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)),
		},
		{
			name: "null month",
			json: "null",
		},
		{
			name:    "invalid format",
			json:    `"2024/03"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m ztime.Month
			err := json.Unmarshal([]byte(tt.json), &m)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.json != "null" {
				assert.Equal(t, tt.want.Format(ztime.MonthFormat), m.Format(ztime.MonthFormat))
			}
		})
	}
}

func TestMonth_Scan(t *testing.T) {
	month := ztime.NewMonth(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC))
	tests := []struct {
		name    string
		input   interface{}
		want    ztime.Month
		wantErr bool
	}{
		{
			name:  "string input",
			input: "2024-03",
			want:  month,
		},
		{
			name:  "time.Time input",
			input: month.Time,
			want:  month,
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
			name:    "invalid month string",
			input:   "invalid month",
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
			var m ztime.Month
			err := m.Scan(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.input != nil {
				assert.Equal(t, tt.want.Format(ztime.MonthFormat), m.Format(ztime.MonthFormat))
			}
		})
	}
}

func TestMonth_UnmarshalJSON_InvalidJSON(t *testing.T) {
	var m ztime.Month
	err := m.UnmarshalJSON([]byte(`{"invalid": "json"}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling ztime.Month: parsing json string")
}

func TestMonth_UnmarshalJSON_InvalidMonthFormat(t *testing.T) {
	var m ztime.Month
	err := m.UnmarshalJSON([]byte(`"invalid month"`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling ztime.Month: parsing")
}

func TestMonth_Value(t *testing.T) {
	now := time.Now()
	month := ztime.NewMonth(now)
	val, err := month.Value()
	assert.NoError(t, err)
	assert.Equal(t, now, val)
}

func TestMonth_Scan_ByteSlice(t *testing.T) {
	var m ztime.Month
	err := m.Scan([]byte("2024-03"))
	assert.NoError(t, err)
	expected := ztime.NewMonth(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, expected.Format(ztime.MonthFormat), m.Format(ztime.MonthFormat))
}

func TestMonth_Prev(t *testing.T) {
	month := ztime.NewMonthFromValues(2024, time.March)
	prev := month.Prev()
	assert.Equal(t, 2024, prev.Year())
	assert.Equal(t, time.February, prev.Month())
}

func TestMonth_Next(t *testing.T) {
	month := ztime.NewMonthFromValues(2024, time.March)
	next := month.Next()
	assert.Equal(t, 2024, next.Year())
	assert.Equal(t, time.April, next.Month())
}

func TestMonth_Add(t *testing.T) {
	month := ztime.NewMonthFromValues(2024, time.March)

	added := month.Add(3)
	assert.Equal(t, 2024, added.Year())
	assert.Equal(t, time.June, added.Month())

	subtracted := month.Add(-2)
	assert.Equal(t, 2024, subtracted.Year())
	assert.Equal(t, time.January, subtracted.Month())
}

func TestMonth_OnDay(t *testing.T) {
	month := ztime.NewMonthFromValues(2024, time.March)
	date := month.OnDay(15)
	assert.Equal(t, 2024, date.Year())
	assert.Equal(t, time.March, date.Month())
	assert.Equal(t, 15, date.Day())
}

func TestMonth_Ptr(t *testing.T) {
	month := ztime.NewMonthFromValues(2024, time.March)
	ptr := month.Ptr()
	assert.NotNil(t, ptr)
	assert.Equal(t, month.Format(ztime.MonthFormat), ptr.Format(ztime.MonthFormat))
}

func TestParseMonth(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ztime.Month
		wantErr bool
	}{
		{
			name:  "valid month",
			input: "2024-03",
			want:  ztime.NewMonthFromValues(2024, time.March),
		},
		{
			name:    "invalid format",
			input:   "2024/03",
			wantErr: true,
		},
		{
			name:    "invalid string",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := ztime.ParseMonth(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want.Format(ztime.MonthFormat), m.Format(ztime.MonthFormat))
		})
	}
}
