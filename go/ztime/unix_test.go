package ztime_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/milagre/zote/go/ztime"
)

func TestNewUnix(t *testing.T) {
	now := time.Now()
	unix := ztime.NewUnix(now, ztime.ResolutionNano)
	assert.Equal(t, now, unix.Time())
	assert.Equal(t, ztime.ResolutionNano, unix.Resolution())
}

func TestUnix_String(t *testing.T) {
	testTime := time.Date(2024, 3, 20, 15, 4, 5, 123456789, time.UTC)
	tests := []struct {
		name     string
		unix     ztime.Unix
		expected string
	}{
		{
			name:     "second resolution",
			unix:     ztime.NewUnix(testTime, ztime.ResolutionSecond),
			expected: "1710947045",
		},
		{
			name:     "millisecond resolution",
			unix:     ztime.NewUnix(testTime, ztime.ResolutionMilli),
			expected: "1710947045123",
		},
		{
			name:     "microsecond resolution",
			unix:     ztime.NewUnix(testTime, ztime.ResolutionMicro),
			expected: "1710947045123456",
		},
		{
			name:     "nanosecond resolution",
			unix:     ztime.NewUnix(testTime, ztime.ResolutionNano),
			expected: "1710947045123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.unix.String())
		})
	}
}

func TestUnix_MarshalJSON(t *testing.T) {
	testTime := time.Date(2024, 3, 20, 15, 4, 5, 123456789, time.UTC)
	tests := []struct {
		name    string
		unix    ztime.Unix
		want    string
		wantErr bool
	}{
		{
			name: "second resolution",
			unix: ztime.NewUnix(testTime, ztime.ResolutionSecond),
			want: "1710947045",
		},
		{
			name: "millisecond resolution",
			unix: ztime.NewUnix(testTime, ztime.ResolutionMilli),
			want: "1710947045123",
		},
		{
			name: "microsecond resolution",
			unix: ztime.NewUnix(testTime, ztime.ResolutionMicro),
			want: "1710947045123456",
		},
		{
			name: "nanosecond resolution",
			unix: ztime.NewUnix(testTime, ztime.ResolutionNano),
			want: "1710947045123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.unix.MarshalJSON()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(data))
		})
	}
}

func TestUnix_UnmarshalJSON(t *testing.T) {
	testTime := time.Date(2024, 3, 20, 15, 4, 5, 123456789, time.UTC)
	tests := []struct {
		name    string
		json    string
		want    ztime.Unix
		wantErr bool
	}{
		{
			name: "second resolution",
			json: "1710947045",
			want: ztime.NewUnix(testTime.Truncate(time.Second), ztime.ResolutionSecond),
		},
		{
			name: "millisecond resolution",
			json: "1710947045123",
			want: ztime.NewUnix(testTime.Truncate(time.Millisecond), ztime.ResolutionMilli),
		},
		{
			name: "microsecond resolution",
			json: "1710947045123456",
			want: ztime.NewUnix(testTime.Truncate(time.Microsecond), ztime.ResolutionMicro),
		},
		{
			name: "nanosecond resolution",
			json: "1710947045123456789",
			want: ztime.NewUnix(testTime, ztime.ResolutionNano),
		},
		{
			name:    "invalid format",
			json:    `"not a number"`,
			wantErr: true,
		},
		{
			name:    "too long",
			json:    "12345678901234567890123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u ztime.Unix
			err := u.UnmarshalJSON([]byte(tt.json))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want.String(), u.String())
		})
	}
}

func TestUnix_SetResolution(t *testing.T) {
	u := ztime.NewUnix(time.Now(), ztime.ResolutionSecond)
	assert.Equal(t, ztime.ResolutionSecond, u.Resolution())

	u.SetResolution(ztime.ResolutionNano)
	assert.Equal(t, ztime.ResolutionNano, u.Resolution())
}
