// Package ztime provides enhanced time types with resolution control,
// JSON marshaling, and SQL support.
//
// # Types
//
// The package provides four specialized time types:
//
//   - Date: Date-only type formatted as YYYY-MM-DD
//   - Time: Time-only type with configurable sub-second precision
//   - Unix: Unix timestamp with configurable resolution
//   - Timezone: Timezone representation wrapping time.Location
//
// # Resolution Control
//
// Time and Unix types support configurable precision via Resolution constants:
// Second, Millisecond, Microsecond, and Nanosecond.
//
// # JSON and SQL Support
//
// All types implement json.Marshaler, json.Unmarshaler, sql.Scanner, and
// driver.Valuer for seamless integration:
//
//	type Event struct {
//		Date      ztime.Date     `json:"date"`       // "2024-01-15"
//		Month     ztime.Month    `json:"month"`      // "2024-01"
//		StartTime ztime.Time     `json:"start_time"` // "14:30:45.000"
//		Created   ztime.Unix     `json:"created_at"` // 1705324845
//		Timezone  ztime.Timezone `json:"timezone"`   // "America/New_York"
//	}
package ztime
