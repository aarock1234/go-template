// Package db wraps pgxpool with sqlc-generated queries, transaction helpers,
// and advisory lock support.
//
//go:generate sqlc generate -f sqlc.yaml
package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"go-template/pkg/db/sqlc"
)

// DB provides a connection pool with query execution and transaction support.
type DB struct {
	pool *pgxpool.Pool
	*sqlc.Queries
}

// New creates a new database connection pool and pings to verify connectivity.
func New(ctx context.Context, url string) (*DB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{
		pool:    pool,
		Queries: sqlc.New(pool),
	}, nil
}

// Close shuts down the connection pool.
func (d *DB) Close() {
	d.pool.Close()
}

// AcquireConn returns a dedicated connection from the pool. The caller
// must call Release on the returned *pgxpool.Conn when done. This is
// useful for session-level operations like advisory locks.
func (d *DB) AcquireConn(ctx context.Context) (*pgxpool.Conn, error) {
	return d.pool.Acquire(ctx)
}

// TryAdvisoryLock attempts to acquire a session-level Postgres advisory
// lock on the given connection. Returns true if the lock was acquired.
func TryAdvisoryLock(ctx context.Context, conn *pgxpool.Conn, key int64) (bool, error) {
	var locked bool
	err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&locked)
	if err != nil {
		return false, fmt.Errorf("pg_try_advisory_lock: %w", err)
	}

	return locked, nil
}

// InTx executes fn inside a database transaction. If fn returns an error
// the transaction is rolled back; otherwise it is committed. Rollback
// failures are logged so the original error is preserved for the caller.
func (d *DB) InTx(ctx context.Context, fn func(sqlc.Querier) error) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(d.WithTx(tx)); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			slog.ErrorContext(ctx, "transaction rollback failed", slog.Any("error", rbErr))
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
