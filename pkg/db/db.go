// Package db wraps pgxpool with sqlc-generated queries, transaction helpers,
// and advisory lock support.
//
//go:generate go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate -f sqlc.yaml
package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aarock1234/go-template/pkg/db/sqlc"
)

// cleanupTimeout bounds a rollback or an unlock that runs after the
// caller's context may already be canceled.
const cleanupTimeout = 5 * time.Second

// DB provides a connection pool with query execution and transaction support.
type DB struct {
	pool *pgxpool.Pool
	*sqlc.Queries
}

// New creates a connection pool for url and pings the database to verify
// connectivity.
func New(ctx context.Context, url string) (*DB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

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

// Ping verifies connectivity to the database.
func (d *DB) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

// TryAdvisoryLock attempts to take the session-level Postgres advisory lock
// for key on a connection dedicated to that lock.
//
// When another session holds the lock, it returns locked == false with a
// nil unlock and a nil error, and holds nothing.
//
// When it acquires the lock, ownership passes to the caller, who must call
// unlock exactly once. unlock releases the lock and returns the connection
// to the pool, using a fresh bounded context so it still works after ctx is
// canceled.
func (d *DB) TryAdvisoryLock(ctx context.Context, key int64) (unlock func() error, locked bool, err error) {
	conn, err := d.pool.Acquire(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("acquire connection: %w", err)
	}

	err = conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&locked)
	if err != nil {
		conn.Release()

		return nil, false, fmt.Errorf("pg_try_advisory_lock: %w", err)
	}

	if !locked {
		conn.Release()

		return nil, false, nil
	}

	unlock = func() error {
		return advisoryUnlock(ctx, conn, key)
	}

	return unlock, true, nil
}

// advisoryUnlock releases the session-level advisory lock for key held by
// conn and returns conn to the pool.
func advisoryUnlock(ctx context.Context, conn *pgxpool.Conn, key int64) error {
	defer conn.Release()

	unlockCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()

	_, err := conn.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", key)
	if err == nil {
		return nil
	}

	err = fmt.Errorf("pg_advisory_unlock: %w", err)

	// The session may still hold the lock. Closing the connection makes
	// Postgres drop the session's locks and makes the pool discard the
	// connection on release instead of handing it to another caller.
	if closeErr := conn.Conn().Close(unlockCtx); closeErr != nil {
		err = errors.Join(err, fmt.Errorf("close connection: %w", closeErr))
	}

	return err
}

// Begin starts a transaction and returns a Querier bound to it along with
// the underlying pgx.Tx. Ownership passes to the caller, who must call
// Commit or Rollback on the transaction.
//
// Prefer InTx for the common case. Use Begin when the transaction must span
// multiple call sites or cannot be expressed as a single callback.
func (d *DB) Begin(ctx context.Context) (sqlc.Querier, pgx.Tx, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}

	return sqlc.New(tx), tx, nil
}

// InTx runs fn inside a transaction and commits it when fn succeeds. The
// transaction is rolled back if fn returns an error or panics, using a
// fresh bounded context so the rollback is still sent after ctx is
// canceled. fn borrows the Querier for the duration of the call and must
// not retain it.
func (d *DB) InTx(ctx context.Context, fn func(sqlc.Querier) error) (err error) {
	q, tx, err := d.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
		defer cancel()

		// After a successful commit, Rollback reports ErrTxClosed.
		rollbackErr := tx.Rollback(rollbackCtx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("rollback transaction: %w", rollbackErr))
		}
	}()

	if err := fn(q); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
