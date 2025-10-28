# zsql - SQL Database Abstraction

Low-level SQL database wrapper with interfaces for core methods and helper methods for common operations.

## Features

- Driver interface for MySQL, PostgreSQL, SQLite3
- Connection pooling and transaction management
- Unified query/exec interface across database types
- Automatic transaction rollback on error
- Database-specific escaping and operators

## Type System

zsql uses a composable interface hierarchy:

```go
// Basic operations
type Queryer interface {
    Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type Executor interface {
    Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Combined operations with driver access
type QueryExecutor interface {
    Queryer
    Executor
    Driver() Driver
}

// Transaction management
type Transactor interface {
    QueryExecutor
    Begin(ctx context.Context, opts *sql.TxOptions) (Transaction, error)
}

type Transaction interface {
    QueryExecutor
    Commit() error
    Rollback() error
}

// Full connection
type Connection interface {
    Transactor
    Close() error
}
```

This means:
- You can pass a `Connection` anywhere a `Queryer`, `Executor`, or `Transactor` is needed
- You can pass a `Transaction` anywhere a `QueryExecutor` is needed
- Helper functions like `zsql.Query()` and `zsql.Exec()` work with any compatible interface

## Driver Interface

The `Driver` interface provides database-specific functionality:

```go
type Driver interface {
    Name() string
    EscapeTable(t string) string
    EscapeColumn(c string) string
    EscapeTableColumn(t string, c string) string
    NullSafeEqualityOperator() string
    EscapeFulltextSearch(search string) string
    PrepareMethod(m string) *string
    IsConflictError(error) bool
}
```

**Available Drivers:**
- **zmysql** - MySQL driver with backtick escaping
- **zsqlite3** - SQLite3 driver with double-quote escaping

## Usage Examples

### Creating a Connection

```go
import (
    "database/sql"

    "github.com/milagre/zote/go/zsql"
    "github.com/milagre/zote/go/zsql/zmysql"
)

db, err := sql.Open("mysql", connectionString)
if err != nil {
    return err
}

conn := zsql.NewConnection(db, zmysql.Driver{})
```

### Transactions

Transactions are provided through the `Begin` helper function which:
- Commits on success (nil return)
- Rolls back on error

```go
err := zsql.Begin(
    ctx,
    db,
    func(ctx context.Context, tx zsql.Transaction) error {
        // Operations within transaction
        _, _, err := zsql.Exec(ctx, tx, "INSERT INTO users (name) VALUES (?)", []any{"Alice"})
        if err != nil {
            return err // Automatic rollback
        }

        return nil // Automatic commit
    },
)
```

### Query

Query provide standard handling for queries that produce results, ensuring all error cases are handled. The callback is called for each found row, allowing you to focus on the actual data. It returns whether or not results were processed, and any error.

```go
found, err := zsql.Query(
    ctx,
    db,
    func(scan zsql.ScanFunc) error {
        var id int64
        var name string
        if err := scan(&id, &name); err != nil {
            return err
        }
        // Process row data
        return nil
    },
    "SELECT id, name FROM users WHERE active = ?",
    []any{true},
)
```

### Execute

Exec provides affected row counts and last insert IDs for every call.

```go
rowsAffected, lastInsertID, err := zsql.Exec(
    ctx,
    db,
    "UPDATE users SET active = ? WHERE id = ?",
    []any{false, 123},
)
```
