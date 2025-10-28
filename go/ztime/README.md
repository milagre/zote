# ztime - Enhanced Time Types

Enhanced time types with resolution control, JSON marshaling, and SQL support.

## Types

- **Date** - Date-only type (YYYY-MM-DD)
- **Time** - Time-only type (HH:MM:SS with optional sub-second precision)
- **Unix** - Unix timestamp with configurable resolution
- **Timezone** - Timezone representation

## Features

- JSON marshaling/unmarshaling with configurable precision
- SQL scanning support (`database/sql` compatible)
- Resolution control (second, millisecond, microsecond, nanosecond)
- String formatting and parsing

## Resolution Control

The `Time` and `Unix` types support configurable precision:

```go
type Resolution int

const (
    Second Resolution = iota
    Millisecond
    Microsecond
    Nanosecond
)
```

## Usage Examples

### Date Type

```go
import "github.com/milagre/zote/go/ztime"

// Create a date
date := ztime.NewDate(2024, 1, 15)

// String representation
str := date.String() // "2024-01-15"

// JSON marshaling
data, err := json.Marshal(date) // "2024-01-15"

// SQL scanning
err := db.QueryRow("SELECT created_date FROM users WHERE id = ?", 1).Scan(&date)

// Convert to time.Time
t := date.At(ztime.NewTime(9, 30, 0))
```

### Time Type

```go
// Create a time with resolution
t := ztime.NewTime(14, 30, 45)
t.SetResolution(ztime.Millisecond)

// String representation
str := t.String() // "14:30:45.000"

// JSON marshaling preserves resolution
data, err := json.Marshal(t) // "14:30:45.000"

// SQL scanning
var timeValue ztime.Time
err := db.QueryRow("SELECT start_time FROM events WHERE id = ?", 1).Scan(&timeValue)

// Convert to time.Time on a specific date
dt := timeValue.On(ztime.NewDate(2024, 1, 15))
```

### Unix Timestamp

```go
// Create from time.Time
now := time.Now()
unix := ztime.NewUnix(now)
unix.SetResolution(ztime.Millisecond)

// String representation includes milliseconds
str := unix.String()

// JSON marshaling as numeric timestamp
data, err := json.Marshal(unix) // 1705324845000 (milliseconds)
```

### Timezone Type

```go
// Create timezone
tz := ztime.NewTimezone("America/New_York")

// JSON marshaling
data, err := json.Marshal(tz) // "America/New_York"

// SQL scanning
var timezone ztime.Timezone
err := db.QueryRow("SELECT user_timezone FROM users WHERE id = ?", 1).Scan(&timezone)
```

## JSON Marshaling

All types implement `json.Marshaler` and `json.Unmarshaler`:

```go
type Event struct {
    Date      ztime.Date     `json:"date"`
    StartTime ztime.Time     `json:"start_time"`
    Created   ztime.Unix     `json:"created_at"`
    Timezone  ztime.Timezone `json:"timezone"`
}

// Marshals to:
// {
//   "date": "2024-01-15",
//   "start_time": "14:30:45.000",
//   "created_at": 1705324845,
//   "timezone": "America/New_York"
// }
```

## SQL Support

The following types implement `sql.Scanner` and `driver.Valuer` for seamless database integration:
- **Date** - Stores as DATE or string in YYYY-MM-DD format
- **Time** - Stores as TIME or string in HH:MM:SS format
- **Timezone** - Stores as VARCHAR/TEXT (timezone name)

```go
type User struct {
    ID        int64
    BirthDate ztime.Date
    StartTime ztime.Time
    Timezone  ztime.Timezone
}

// Scan from database
var user User
err := db.QueryRow(
    "SELECT id, birth_date, start_time, timezone FROM users WHERE id = ?",
    1,
).Scan(&user.ID, &user.BirthDate, &user.StartTime, &user.Timezone)

// Insert/Update to database
_, err = db.Exec(
    "INSERT INTO users (birth_date, start_time, timezone) VALUES (?, ?, ?)",
    user.BirthDate,
    user.StartTime,
    user.Timezone,
)
```