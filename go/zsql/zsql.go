package zsql

import (
	"context"
	"database/sql"
	"fmt"
)

type Driver struct {
	name string
}

func NewDriver(name string) Driver {
	return Driver{
		name: name,
	}
}

func (d Driver) Name() string {
	return d.name
}

type Connection struct {
	*sql.DB
	driver Driver
}

func NewConnection(db *sql.DB, driver Driver) *Connection {
	return &Connection{
		DB:     db,
		driver: driver,
	}
}

func (c Connection) Driver() string {
	return c.driver.name
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

func Begin(ctx context.Context, db *sql.DB, cb func(ctx context.Context, x *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
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
