package db

import (
	"context"
	"database/sql"
)

type Database interface {
	BeginTransaction(ctx context.Context) (*sql.Tx, error)
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row
	Execute(ctx context.Context, query string, args ...any) (sql.Result, error)
	Close() error
}
