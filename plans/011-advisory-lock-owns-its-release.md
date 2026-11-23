# Plan 011: Taking an advisory lock returns the function that releases it

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 3112ea3..HEAD -- pkg/db/db.go`
> If the file changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch,
> treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW (no caller exists in the repository)
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `3112ea3`, 2026-11-23

## Why this matters

`pkg/db` offers session-level PostgreSQL advisory locks through two separate
pieces: `AcquireConn`, whose documentation tells the caller only to call
`Release` on the connection, and `TryAdvisoryLock`, which takes the lock on
that connection. There is no unlock function. A session-level advisory lock
belongs to the database session, not to the transaction or the checkout, so
a caller who follows the documentation returns the connection to the pool
with the lock still held. Other processes stay locked out until the pool
happens to recycle that connection, and a later caller who draws the same
connection is told it acquired a lock that was never released. After this
plan, acquiring the lock hands back the one function that releases it
correctly, so the API cannot be used wrongly.

## Current state

- `pkg/db/db.go:19-21`:

  ```go
  // rollbackTimeout bounds a rollback that runs after the caller's context
  // may already be canceled.
  const rollbackTimeout = 5 * time.Second
  ```

- `pkg/db/db.go:59-77`:

  ```go
  // AcquireConn returns a dedicated connection from the pool. The caller
  // must call Release on the returned *pgxpool.Conn when done. This is
  // useful for session-level operations like advisory locks.
  func (d *DB) AcquireConn(ctx context.Context) (*pgxpool.Conn, error) {
  	return d.pool.Acquire(ctx)
  }

  // TryAdvisoryLock attempts to acquire a session-level Postgres advisory
  // lock on the given connection. It reports whether the lock was acquired.
  func TryAdvisoryLock(ctx context.Context, conn *pgxpool.Conn, key int64) (bool, error) {
  	var locked bool

  	err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&locked)
  	if err != nil {
  		return false, fmt.Errorf("pg_try_advisory_lock: %w", err)
  	}

  	return locked, nil
  }
  ```

- The pattern to follow for cleanup that must run after cancellation is the
  rollback in `InTx`, `pkg/db/db.go:105-114`:

  ```go
  	defer func() {
  		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
  		defer cancel()

  		// After a successful commit, Rollback reports ErrTxClosed.
  		rollbackErr := tx.Rollback(rollbackCtx)
  		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
  			err = errors.Join(err, fmt.Errorf("rollback transaction: %w", rollbackErr))
  		}
  	}()
  ```

- Neither `AcquireConn` nor `TryAdvisoryLock` has a caller anywhere in the
  repository. Confirm with
  `grep -rn "AcquireConn\|TryAdvisoryLock" --include=*.go .` → matches only
  in `pkg/db/db.go`.

- The package comment (`pkg/db/db.go:1-2`) says the package provides
  "advisory lock support", which stays true.

- The `pgx` version is `github.com/jackc/pgx/v5 v5.11.0` (`go.mod:10`).

Conventions that apply (from the repository's `AGENTS.md` and `STYLE.md`):

- Every acquired resource has an explicit owner and outcome.
- Every nonempty struct literal is multiline with keyed fields.
- Errors are wrapped with `fmt.Errorf("context: %w", err)`; independent
  failures are combined with `errors.Join`.
- Doc comments state the contract, including who must call what.

## Commands you will need

A local, git-ignored `go.work` may exist on the maintainer's machine. Prefix
Go commands with `GOWORK=off` so results match CI.

| Purpose      | Command                                                                                | Expected on success |
|--------------|----------------------------------------------------------------------------------------|---------------------|
| Build        | `GOWORK=off go build ./...`                                                            | exit 0, no output   |
| Vet          | `GOWORK=off go vet ./...`                                                              | exit 0, no output   |
| Format check | `gofmt -l pkg/db`                                                                      | no output           |
| Lint         | `GOWORK=off go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run` | exit 0, `0 issues.` |
| Find pgx     | `GOWORK=off go list -m -f '{{.Dir}}' github.com/jackc/pgx/v5`                          | prints the module directory |

Run lint once before changing anything to learn the baseline; run it in the
background.

## Scope

**In scope** (the only files you should modify):

- `pkg/db/db.go`

**Out of scope** (do NOT touch, even though they look related):

- `pkg/db/sqlc/` — generated code.
- `Begin` and `InTx`, beyond the shared timeout constant described in Step 1.
- `README.md` — its one-line description of `pkg/db` remains accurate.
- Blocking lock acquisition (`pg_advisory_lock`) and transaction-scoped
  locks. Only the existing try-lock is reshaped.
- Any test file. This repository forbids adding or running tests unless the
  maintainer asks for them.
- Uncommitted maintainer work in `README.md`, `AGENTS.md`, `STYLE.md`,
  `pkg/service/service.go`, and `.agents/skills/go-style/`. Do not stash,
  revert, commit, or reformat those files.

## Git workflow

- Branch: `advisor/011-advisory-lock-owns-its-release`
- One commit. Message style is a short imperative sentence, as in `git log`:
  "Use fphttp v2 profiles in the client". Suggested:
  `Return an unlock function from TryAdvisoryLock`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Share the cleanup timeout

Rename `rollbackTimeout` to `cleanupTimeout` and reword its comment so it
covers both uses: it bounds a rollback or an unlock that runs after the
caller's context may already be canceled. Update the one existing use in
`InTx`.

**Verify**: `grep -n "rollbackTimeout" pkg/db/db.go` → no matches;
`GOWORK=off go build ./pkg/db` → exit 0.

### Step 2: Confirm how pgxpool treats a closed connection

Find the pgx module directory with the "Find pgx" command, then read
`pgxpool/conn.go` there. Confirm two facts about `(*pgxpool.Conn).Release`:

1. It does not run `pg_advisory_unlock_all` or otherwise reset session
   state on a healthy connection.
2. A connection that has been closed is destroyed instead of being returned
   to the pool (look for a check such as `conn.IsClosed()` followed by
   `Destroy`).

**Verify**: both facts hold. If either does not, STOP.

### Step 3: Replace the two functions with one method

Delete `AcquireConn` and the package-level `TryAdvisoryLock`. Add one method
on `*DB` in their place:

```go
func (d *DB) TryAdvisoryLock(ctx context.Context, key int64) (unlock func() error, locked bool, err error)
```

Contract, which the doc comment must state:

- It attempts to take the session-level advisory lock for `key` on a
  connection dedicated to that lock.
- When another session holds the lock it returns `locked == false`, a nil
  `unlock`, and a nil error, and holds nothing.
- When it acquires the lock, the caller must call `unlock` exactly once.
  `unlock` releases the lock and returns the connection to the pool. It
  still works after `ctx` is canceled.

Behavior:

- Acquire a connection from `d.pool`. On failure return a wrapped error.
- Run `SELECT pg_try_advisory_lock($1)`. On a query error, or when the lock
  was not acquired, release the connection before returning.
- `unlock` uses a fresh bounded context, exactly like the rollback in `InTx`:
  `context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)`. It runs
  `SELECT pg_advisory_unlock($1)` on the same connection and then releases
  the connection.
- If the unlock query fails, the session may still hold the lock, so the
  connection must not go back to the pool usable: close the underlying
  connection (`conn.Conn().Close(...)`) before releasing it, which makes the
  pool discard it and makes PostgreSQL drop the session's locks. Return the
  unlock error joined with any close error.

Keep the method readable: if the closure grows past a screenful, move its
body into a small unexported function that takes the connection and key.
Add a short comment where the connection is closed explaining why.

**Verify**: `GOWORK=off go build ./...` → exit 0;
`grep -n "func (d \*DB) AcquireConn\|^func TryAdvisoryLock" pkg/db/db.go` →
no matches; `grep -n "pg_advisory_unlock" pkg/db/db.go` → at least one match.

### Step 4: Full verification

Run build, vet, format check, and lint from the commands table.

**Verify**: each command gives its expected result.

## Test plan

No tests. This repository's rules forbid adding, running, or proposing tests
unless the maintainer explicitly asks. Verification is the build, vet,
format, and lint commands, the pgxpool source check in Step 2, and a
top-to-bottom read of `pkg/db/db.go`.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `GOWORK=off go build ./...` and `GOWORK=off go vet ./...` exit 0
- [ ] `gofmt -l pkg/db` prints nothing
- [ ] Lint reports no issue that was not in the baseline
- [ ] `grep -rn "AcquireConn" --include=*.go .` returns no matches
- [ ] `grep -n "pg_advisory_unlock" pkg/db/db.go` matches
- [ ] `grep -n "rollbackTimeout" pkg/db/db.go` returns no matches
- [ ] `git diff --stat` lists only `pkg/db/db.go`
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- `pkg/db/db.go` does not match the excerpts.
- `AcquireConn` or `TryAdvisoryLock` has a caller outside `pkg/db/db.go`.
- Step 2 shows that pgxpool resets session state on release, or does not
  discard closed connections. The design in Step 3 depends on both facts.
- Lint rejects the three-value return or the named results. Report the
  message; do not add a `//nolint` comment.
- A verification fails twice after a reasonable fix attempt.

## Maintenance notes

- This removes two exported identifiers. Nothing in the repository uses
  them, and the template has no compatibility promise for `pkg/db`, but
  projects already scaffolded from an older template are unaffected either
  way because they own their copy.
- Session-level advisory locks are re-entrant within a session. Because each
  call now uses its own dedicated connection for the life of the lock, two
  holders in one process can no longer silently stack on a shared
  connection.
- Reviewers should read the failure path of `unlock` closely: it is the one
  place where a lock could otherwise leak into the pool.
- A caller that only needs the lock for the length of one transaction can
  use `pg_try_advisory_xact_lock` inside `InTx` instead; that was not added
  here.

Standards: Meet the solution standards in ~/.agents/AGENTS.md: idiomatic, beautiful, elegant, seamless, well-structured, dynamic, and future-proof, as explained in ~/.agents/CODING.md.
Guidance: Before implementing, read the repository's AGENTS.md and STYLE.md (if present in your checkout), load the go-style skill (`.agents/skills/go-style/SKILL.md`), and read `pkg/db/db.go` in full.
Review: Read every changed file from top to bottom for readability before declaring the plan done.
Evidence: Verify with build, vet, formatting checks, lint, command output, and the relevant file contents. Do not add or run tests unless tests are explicitly requested.
Execution: Run every shell command that may take more than a few seconds, and every subagent task, in the background, and keep working on what does not depend on it until its completion notification arrives. Never sleep or poll.
