package zsql

import (
	"context"
	"database/sql"

	"github.com/milagre/zote/go/zlog"
)

type LoggingTransactor struct {
	Transactor
}

func (q LoggingTransactor) Begin(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	logger := zlog.FromContext(ctx)
	logger.Infof("SQL|Begin")

	tx, err := q.Transactor.Begin(ctx, opts)
	return LoggingTransaction{tx}, err
}

func (q LoggingTransactor) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	logger := zlog.FromContext(ctx)
	logger.Infof("SQL|Q: %s | %v", query, args)

	return q.Transactor.Query(ctx, query, args...)
}

func (q LoggingTransactor) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	logger := zlog.FromContext(ctx)
	logger.Infof("SQL|Q: %s | %v", query, args)

	return q.Transactor.Exec(ctx, query, args...)
}

type LoggingTransaction struct {
	Transaction
}

func (q LoggingTransaction) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	logger := zlog.FromContext(ctx)
	logger.Infof("SQL|Q: %s | %v", query, args)

	return q.Transaction.Query(ctx, query, args...)
}

func (q LoggingTransaction) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	logger := zlog.FromContext(ctx)
	logger.Infof("SQL|Q: %s | %v", query, args)

	return q.Transaction.Exec(ctx, query, args...)
}
