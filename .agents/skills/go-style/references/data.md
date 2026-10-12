# Database work and data ownership

[Back to the Go entrypoint](../SKILL.md). Read this topic when changing database operations. Use the project's existing query layer and driver; the template uses pgx v5 and generated sqlc queries, while the standard library's `database/sql` has different lifetime APIs.

- [Transaction Ownership](#go-transactions)
- [Rows and Query Contracts](#go-query-rows)

<a id="go-transactions"></a>

## Transaction Ownership

The scope that begins a transaction owns commit and rollback unless it explicitly transfers that ownership to its caller, as the template's `DB.Begin` does. Give a multi-step operation one transaction-bound query capability; do not mix pool/database calls into the same unit of work. Return success only after commit succeeds, and preserve the operation's error when cleanup also fails.

The following helper illustrates pgx v5 ownership. Its callback uses the transaction passed to it; the template's sqlc adapter can bind generated queries to that transaction.

The callback borrows the transaction for its call. It must not commit, roll back, retain the transaction, or leave transaction work running after it returns.

```go
func InTx(
	ctx context.Context,
	pool *pgxpool.Pool,
	apply func(pgx.Tx) error,
) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rollbackErr := tx.Rollback(cleanupCtx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("rollback transaction: %w", rollbackErr))
		}
	}()

	if err := apply(tx); err != nil {
		return fmt.Errorf("apply transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
```

Imports for this excerpt are `context`, `errors`, `fmt`, `time`, `github.com/jackc/pgx/v5`, and `github.com/jackc/pgx/v5/pgxpool`. `errors.Join` needs Go 1.20+. Make the five-second cleanup budget an application policy when this helper is adopted; it is not a universal database timeout.

pgx's begin context controls the begin statement; canceling it does not automatically roll back the transaction. Subsequent transaction methods take their own context, and a canceled request context may prevent rollback from being sent. A fresh bounded cleanup context lets the owner attempt release. `BeginFunc`/`BeginTxFunc` can reduce boilerplate when their context and error behavior fit the operation.

With `database/sql`, use `BeginTx`, defer `Rollback()`, and inspect `Commit()`. Its begin context remains associated with the transaction through finalization, and cancellation causes the package to roll it back. Do not copy pgx's method signatures or cancellation semantics into a standard-library helper.

Keep slow external calls and user interaction outside the transaction when possible. A database transaction does not atomically commit an HTTP call or queue publish; persist a handoff/outbox when the domain requires that guarantee. Retry only a classified retryable transaction failure with a bounded whole-operation policy. An ambiguous commit outcome is not permission to repeat a non-idempotent write.

Related: [cleanup errors](errors.md#go-close-errors), [context boundaries](context.md), [retry safety](concurrency.md#go-retry-contracts).

Sources: [Go transaction ownership](https://go.dev/doc/database/execute-transactions), [pgx transaction contracts](https://pkg.go.dev/github.com/jackc/pgx/v5#Tx), [pgx BeginFunc](https://pkg.go.dev/github.com/jackc/pgx/v5#BeginFunc).


<a id="go-query-rows"></a>

## Rows and Query Contracts

Own every rows cursor until it is consumed or closed. Handle the query error, each scan error, and the terminal iteration error; `Next()` returning false can mean failure rather than successful exhaustion. Do not return partial results as success when the contract promises a complete list.

This standard-library example assumes a PostgreSQL driver (`$1` placeholders) and a non-nil empty list on successful exhaustion. It is a query-layer excerpt, not a replacement for the template's generated queries.

```go
type UserSummary struct {
	ID   string
	Name string
}

func ListActiveUsers(
	ctx context.Context,
	db *sql.DB,
	limit int,
) (users []UserSummary, err error) {
	if limit < 1 {
		return nil, errors.New("limit must be positive")
	}

	rows, err := db.QueryContext(ctx, `
		SELECT id, name
		FROM users
		WHERE active = true
		ORDER BY id
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query active users: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			users = nil
			err = errors.Join(err, fmt.Errorf("close active user rows: %w", closeErr))
		}
	}()

	users = make([]UserSummary, 0)

	for rows.Next() {
		var user UserSummary
		if err := rows.Scan(&user.ID, &user.Name); err != nil {
			return nil, fmt.Errorf("scan active user: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active users: %w", err)
	}

	return users, nil
}
```

Imports are `context`, `database/sql`, `errors`, and `fmt`. Successful exhaustion closes standard-library rows automatically; deferred close also covers early returns. `Rows.Err` remains necessary. With pgx v5, `Rows.Close()` returns no error, and `CollectRows` owns closing; use the actual driver's contract rather than forcing this close-error pattern onto it.

Pass data as parameters. A placeholder cannot safely select an SQL identifier; choose a column/order from a closed domain mapping or the driver's identifier API. Distinguish not-found from an empty list and from a database failure. Keep ordering explicit when pagination or repeatability needs it, and avoid loading an unbounded table merely to filter it in memory.

Related: [collection contracts](types.md#go-collection-contracts), [typed boundaries](types.md#go-json-boundaries), [resource cleanup](errors.md#go-close-errors).

Sources: [database/sql rows](https://pkg.go.dev/database/sql#Rows), [query parameters](https://go.dev/doc/database/querying), [pgx row collection](https://pkg.go.dev/github.com/jackc/pgx/v5#CollectRows).

