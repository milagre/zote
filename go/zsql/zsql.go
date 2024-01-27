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
}

type Transactor interface {
	Queryer
	TransactionBeginner

	Driver() Driver
}

type Transaction interface {
	Queryer
	TransactionEnder
}

type TransactionBeginner interface {
	Begin(ctx context.Context, opts *sql.TxOptions) (Transaction, error)
}

type TransactionEnder interface {
	Rollback() error
	Commit() error
}

type Queryer interface {
	Selector
	Executor
}

type Selector interface {
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type Executor interface {
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Deprecated: Use selector
type Queryable interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type Connection struct {
	db     *sql.DB
	driver Driver
}

var _ interface {
	Transactor
} = Connection{}

func NewConnection(db *sql.DB, driver Driver) Connection {
	return Connection{
		db:     db,
		driver: driver,
	}
}

func (c Connection) Driver() Driver {
	return c.driver
}

func (c Connection) Begin(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	tx, err := c.db.BeginTx(ctx, opts)
	return transaction{tx: tx}, err
}

func (c Connection) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, query, args...)
}

func (c Connection) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return c.db.ExecContext(ctx, query, args...)
}

func (c Connection) Close() error {
	return c.db.Close()
}

type transaction struct {
	tx *sql.Tx
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

type ScanFunc func(dest ...any) error
type QueryCallback func(ScanFunc) error

func Query(ctx context.Context, db Selector, cb QueryCallback, query string, args []any) (bool, error) {
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
