package ztime_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/milagre/zote/go/ztime"
)

func TestNewTimezone(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	assert.NoError(t, err)

	tz := ztime.NewTimezone(loc)
	assert.Equal(t, loc, tz.Location)
}

func TestTimezone_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		timezone ztime.Timezone
		want     string
		wantErr  bool
	}{
		{
			name:     "UTC",
			timezone: ztime.NewTimezone(time.UTC),
			want:     `"UTC"`,
		},
		{
			name: "America/New_York",
			timezone: func() ztime.Timezone {
				loc, _ := time.LoadLocation("America/New_York")
				return ztime.NewTimezone(loc)
			}(),
			want: `"America/New_York"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.timezone)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(data))
		})
	}
}

func TestTimezone_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    string
		wantErr bool
	}{
		{
			name: "UTC",
			json: `"UTC"`,
			want: "UTC",
		},
		{
			name: "null timezone",
			json: "null",
		},
		{
			name:    "invalid timezone",
			json:    `"Invalid/Zone"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tz ztime.Timezone
			err := json.Unmarshal([]byte(tt.json), &tz)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.json != "null" {
				assert.Equal(t, tt.want, tz.String())
			}
		})
	}
}

func TestTimezone_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr bool
	}{
		{
			name:  "string input",
			input: "UTC",
			want:  "UTC",
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
			name:    "time.Time input",
			input:   time.Now(),
			wantErr: true,
		},
		{
			name:  "nil input",
			input: nil,
		},
		{
			name:    "invalid timezone string",
			input:   "Invalid/Timezone",
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
			var tz ztime.Timezone
			err := tz.Scan(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.input != nil {
				assert.Equal(t, tt.want, tz.String())
			}
		})
	}
}

func TestTimezone_Scan_ByteSlice(t *testing.T) {
	var tz ztime.Timezone
	err := tz.Scan([]byte("UTC"))
	assert.NoError(t, err)
	assert.Equal(t, "UTC", tz.String())
}

func TestTimezone_Value(t *testing.T) {
	loc, err := time.LoadLocation("UTC")
	assert.NoError(t, err)

	tz := ztime.NewTimezone(loc)
	val, err := tz.Value()
	assert.NoError(t, err)
	assert.Equal(t, "UTC", val)
}

func TestTimezone_UnmarshalJSON_InvalidJSON(t *testing.T) {
	var tz ztime.Timezone
	err := tz.UnmarshalJSON([]byte(`{"invalid": "json"}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling timezone: parsing value into string")
}
