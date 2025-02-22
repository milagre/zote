package zsql

import (
	"context"
	"database/sql"

	"github.com/milagre/zote/go/zlog"
)

type LoggingConnection struct {
	Connection
}

func NewLoggingConnection(c Connection) Connection {
	return LoggingConnection{c}
}

func (q LoggingConnection) Begin(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	tx, err := logBegin(ctx, opts, q.Connection)

	return LoggingTransaction{tx, zlog.FromContext(ctx)}, err
}

func (q LoggingConnection) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return logQuery(ctx, query, args, q.Connection)
}

func (q LoggingConnection) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return logExec(ctx, query, args, q.Connection)
}

type LoggingTransactor struct {
	Transactor
}

func (q LoggingTransactor) Begin(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	tx, err := logBegin(ctx, opts, q.Transactor)

	return LoggingTransaction{tx, zlog.FromContext(ctx)}, err
}

func (q LoggingTransactor) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return logQuery(ctx, query, args, q.Transactor)
}

func (q LoggingTransactor) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return logExec(ctx, query, args, q.Transactor)
}

type LoggingTransaction struct {
	Transaction

	logger zlog.Logger
}

func (q LoggingTransaction) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return logQuery(ctx, query, args, q.Transaction)
}

func (q LoggingTransaction) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return logExec(ctx, query, args, q.Transaction)
}

func (q LoggingTransaction) Commit() error {
	return logCommit(q.logger, q.Transaction)
}

func (q LoggingTransaction) Rollback() error {
	return logRollback(q.logger, q.Transaction)
}

// Implementations

func logBegin(ctx context.Context, opts *sql.TxOptions, t TransactionBeginner) (Transaction, error) {
	logger := zlog.FromContext(ctx)
	logger.Debugf("SQL|Begin")

	return t.Begin(ctx, opts)
}

func logQuery(ctx context.Context, query string, args []any, q Queryer) (*sql.Rows, error) {
	logger := zlog.FromContext(ctx)
	logger.Debugf("SQL|Q: %s | %v", query, args)

	return q.Query(ctx, query, args...)
}

func logExec(ctx context.Context, query string, args []any, e Executor) (sql.Result, error) {
	logger := zlog.FromContext(ctx)
	logger.Debugf("SQL|E: %s | %v", query, args)

	return e.Exec(ctx, query, args...)
}

func logCommit(logger zlog.Logger, t Transaction) error {
	logger.Debugf("SQL|Commit")

	return t.Commit()
}

func logRollback(logger zlog.Logger, t Transaction) error {
	logger.Debugf("SQL|Rollback")

	return t.Rollback()
}
