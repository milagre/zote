# zorm - ORM Framework

ORM providing CRUD operations with generic type support.

## Features

- Repository pattern with `Find`, `Get`, `Put`, `Delete` operations
- Relation loading with `Include` system
- Generic type support: `zorm.Find[T]()`, `zorm.Get[T]()`
- Clause-based query building via `zelement/zclause`
- Transaction support via `zsql`

## Usage with zormsql

The main implementation is in the `zormsql` subpackage, which provides SQL-based ORM operations.

```go
package main

import (
    "context"
    "database/sql"
    "time"

    "github.com/milagre/zote/go/zorm"
    "github.com/milagre/zote/go/zorm/zormsql"
    "github.com/milagre/zote/go/zsql"
    "github.com/milagre/zote/go/zsql/zmysql"
    "github.com/milagre/zote/go/zelement/zclause"
    "github.com/milagre/zote/go/zelement/zsort"
)

// Define your model
type User struct {
    ID       int64
    Created  time.Time
    Modified *time.Time
    Name     string
    Email    string
    Active   bool
}

// Define the mapping
var userMapping = zormsql.Mapping{
    PtrType:    &User{},
    Table:      "users",
    PrimaryKey: []string{"id"},
    Columns: []zormsql.Column{
        {Name: "id", Field: "ID", NoInsert: true, NoUpdate: true},
        {Name: "created", Field: "Created", NoInsert: true, NoUpdate: true},
        {Name: "modified", Field: "Modified", NoInsert: true, NoUpdate: true},
        {Name: "name", Field: "Name"},
        {Name: "email", Field: "Email"},
        {Name: "active", Field: "Active"},
    },
}

func main() {
    ctx := context.Background()

    // Setup database connection
    db, _ := sql.Open("mysql", "user:pass@tcp(localhost:3306)/mydb")
    conn := zsql.NewConnection(db, zmysql.Driver{})

    // Create repository and register mapping
    repo := zormsql.NewRepository("myapp", conn)
    repo.AddMapping(userMapping)

    // INSERT - Put with empty ID inserts new record
    newUser := &User{Name: "Alice", Email: "alice@example.com", Active: true}
    zorm.Put(ctx, repo, []*User{newUser}, zorm.PutOptions{})
    // newUser.ID is now populated with auto-generated ID

    // UPDATE - Put with existing ID updates record
    newUser.Name = "Alice Smith"
    zorm.Put(ctx, repo, []*User{newUser}, zorm.PutOptions{})

    // FIND - Query multiple records
    var activeUsers []*User
    zorm.Find(ctx, repo, &activeUsers, zorm.FindOptions{
        Where: zclause.Eq("active", true),
        Sort: []zsort.Sort{{Field: "Created", Desc: true}},
    })

    // GET - Fetch specific records by primary key
    user := &User{ID: newUser.ID}
    zorm.Get(ctx, repo, []*User{user}, zorm.GetOptions{})

    // DELETE - Remove records
    zorm.Delete(ctx, repo, []*User{user}, zorm.DeleteOptions{})

    // TRANSACTION - Wrap operations in transaction
    tx, _ := repo.Begin(ctx)
    zorm.Put(ctx, tx, []*User{{Name: "Bob"}}, zorm.PutOptions{})
    tx.Commit() // or tx.Rollback()
}
```

## Mapping Configuration

**Column Options:**
- `NoInsert: true` - Exclude from INSERT (auto-generated fields)
- `NoUpdate: true` - Exclude from UPDATE (timestamps, immutable fields)

**Primary Key:**
- Single column: `PrimaryKey: []string{"id"}`
- Composite: `PrimaryKey: []string{"user_id", "role_id"}`

**Relations:**
Define relationships in the mapping for eager loading (see `Relation` type in zormsql)
