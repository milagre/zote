# zcache - Cache Abstraction

Cache abstraction with read-through cache algorithm.

## Cache Interface

```go
type Cache interface {
    Set(ctx context.Context, namespace string, key string, expiration time.Duration, value []byte) error
    Get(ctx context.Context, namespace string, key string) (<-chan []byte, error)
}
```

## Read-Through Pattern

The read-through pattern automatically handles:
1. Cache lookup
  a. On miss:
    i. Fetch from source
    ii. Cache the result
    iii. Return the result
  b. On hit
    i. Parse data in cache
    ii. Return result

Cache failures are non-fatal and return warnings while still serving data from source.

## Usage Example

```go
import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    "github.com/milagre/zote/go/zcache"
    "github.com/milagre/zote/go/zwarn"
)

type UserData struct {
    ID   int64
    Name string
}

func FetchUser(ctx context.Context, cache zcache.Cache, userID int64) (UserData, zwarn.Warning, error) {
    return zcache.ReadThrough(
        ctx,
        cache,
        "users",                           // namespace
        fmt.Sprintf("user:%d", userID),    // key
        time.Hour,                         // expiration

        // Loader function (called on cache miss)
        func(ctx context.Context) (UserData, error) {
            return database.FetchUser(ctx, userID)
        },

        // Marshal function (called on loaded data to store in cache)
        func(u UserData) ([]byte, error) {
            return json.Marshal(u)
        },

        // Unmarshal function (called for cached data)
        func(data []byte) (UserData, error) {
            var user UserData
            err := json.Unmarshal(data, &user)
            return user, err
        },
    )
}
```

## Warning Handling

The readthrough method returns three values:
- **Result** - The parsed data (either from cache or source)
- **Warning** - Non-fatal issues (cache failures, parse errors on cached data)
- **Error** - Fatal errors that prevent data retrieval

```go
user, warnings, err := FetchUser(ctx, cache, 123)
if err != nil {
    return err // Fatal error
}
if warnings != nil {
    log.Warn(warnings) // Log cache issues but continue
}
// Use user data normally
```

## Cache Failure Behavior

When cache operations fail:
1. **Get failure**: Loader is called to fetch from source
2. **Parse failure**: Loader is called to fetch fresh data from source
3. **Set failure**: Data is returned successfully but warning is issued

This ensures that cache issues never prevent application functionality.