// Package zsql provides a SQL database abstraction layer with composable
// interfaces for database operations.
//
// zsql wraps database/sql with driver-specific functionality for MySQL,
// PostgreSQL, and SQLite3, providing connection pooling, transaction
// management, and a unified query/exec interface.
//
// # Interface Hierarchy
//
// zsql adds interfaces that database/sql is missing:
//
//   - Connection implements Transactor, QueryExecutor, Queryer, Executor
//   - Transaction implements QueryExecutor, Queryer, Executor
//
// This enables helper functions like Query and Exec to work uniformly
// with either connections or transactions.
//
// # Creating a Connection
//
//	import (
//		"database/sql"
//
//		"github.com/milagre/zote/go/zsql"
//		"github.com/milagre/zote/go/zsql/zmysql"
//	)
//
//	db, err := sql.Open("mysql", connectionString)
//	if err != nil {
//		return err
//	}
//
//	conn := zsql.NewConnection(db, zmysql.Driver{})
//
// # Transactions
//
// The Begin helper manages transaction lifecycle, committing on success
// and rolling back on error:
//
//	err := zsql.Begin(ctx, db, func(ctx context.Context, tx zsql.Transaction) error {
//		_, _, err := zsql.Exec(ctx, tx, "INSERT INTO users (name) VALUES (?)", []any{"Alice"})
//		if err != nil {
//			return err // automatic rollback
//		}
//		return nil // automatic commit
//	})
//
// # Querying
//
// Query handles result iteration and error checking, calling the callback
// for each row:
//
//	found, err := zsql.Query(ctx, db, func(scan zsql.ScanFunc) error {
//		var id int64
//		var name string
//		if err := scan(&id, &name); err != nil {
//			return err
//		}
//		// process row
//		return nil
//	}, "SELECT id, name FROM users WHERE active = ?", []any{true})
//
// # Executing
//
// Exec returns affected row count and last insert ID:
//
//	rowsAffected, lastInsertID, err := zsql.Exec(ctx, db,
//		"UPDATE users SET active = ? WHERE id = ?", []any{false, 123})
//
// # Drivers
//
// Use database-specific drivers for proper escaping and dialect handling. See zorm
//
//   - zmysql.Driver - MySQL with backtick escaping
//   - zsqlite3.Driver - SQLite3 with double-quote escaping
package zsql

import (
	"context"
	"database/sql"
	"fmt"
)

type Driver interface {
	Name() string
	EscapeTable(t string) string
	EscapeColumn(c string) string
	EscapeTableColumn(t string, c string) string
	NullSafeEqualityOperator() string

	PrepareMethod(m string) *string

	IsConflictError(error) bool
}

type HasDriver interface {
	Driver() Driver
}

type Queryer interface {
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type Executor interface {
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type QueryExecutor interface {
	Queryer
	Executor
	Driver() Driver
}

type Transactor interface {
	QueryExecutor
	TransactionBeginner
}

type Transaction interface {
	QueryExecutor
	TransactionEnder
}

type TransactionBeginner interface {
	Begin(ctx context.Context, opts *sql.TxOptions) (Transaction, error)
}

type TransactionEnder interface {
	Rollback() error
	Commit() error
}

type Connection interface {
	Transactor
	Close() error
}

type connection struct {
	db     *sql.DB
	driver Driver
}

func NewConnection(db *sql.DB, driver Driver) Connection {
	return connection{
		db:     db,
		driver: driver,
	}
}

func (c connection) Driver() Driver {
	return c.driver
}

func (c connection) Begin(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	tx, err := c.db.BeginTx(ctx, opts)
	return transaction{
		source: c,
		tx:     tx,
	}, err
}

func (c connection) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, query, args...)
}

func (c connection) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return c.db.ExecContext(ctx, query, args...)
}

func (c connection) Close() error {
	return c.db.Close()
}

type transaction struct {
	source connection
	tx     *sql.Tx
}

func (t transaction) Driver() Driver {
	return t.source.Driver()
}

func (t transaction) Commit() error {
	return t.tx.Commit()
}

func (t transaction) Rollback() error {
	return t.tx.Rollback()
}

func (t transaction) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

func (t transaction) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

type Options map[string]interface{}

func (o Options) Merge(opts Options) Options {
	res := Options{}
	for k, v := range o {
		res[k] = v
	}
	for k, v := range opts {
		res[k] = v
	}
	return res
}

func (o Options) ToStringMapString() map[string]string {
	res := map[string]string{}
	for k, v := range o {
		res[k] = fmt.Sprintf("%v", v)
	}
	return res
}

func Begin(ctx context.Context, db Transactor, cb func(context.Context, Transaction) error) error {
	tx, err := db.Begin(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}

	err = cb(ctx, tx)
	if err != nil {
		e := tx.Rollback()
		if e != nil {
			return fmt.Errorf("rolling back transaction after error: %s: %w", e, err)
		}

		return fmt.Errorf("transaction rolled back: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commiting transaction: %w", err)
	}

	return nil
}

type (
	ScanFunc      func(dest ...any) error
	QueryCallback func(ScanFunc) error
)

func Query(ctx context.Context, db Queryer, cb QueryCallback, query string, args []any) (bool, error) {
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("executing query: %w", err)
	}

	defer func() {
		if err = rows.Close(); err != nil {
			err = fmt.Errorf("closing rows: %w", err)
		}
	}()

	found := false
	for rows.Next() {
		found = true
		if err = cb(rows.Scan); err != nil {
			return false, fmt.Errorf("scanning row: %w", err)
		}
	}

	if err = rows.Err(); err != nil {
		return false, fmt.Errorf("processing rows: %w", err)
	}

	return found, err
}

func Exec(ctx context.Context, db Executor, query string, args []any) (int, int64, error) {
	res, err := db.Exec(ctx, query, args...)
	if err != nil {
		return 0, 0, fmt.Errorf("executing query: %w", err)
	}

	count, err := res.RowsAffected()
	if err != nil {
		return 0, 0, fmt.Errorf("getting affected rows: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, 0, fmt.Errorf("getting last insert id: %w", err)
	}

	return int(count), id, nil
}
