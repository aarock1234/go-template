# Plan 010: A second interrupt during shutdown terminates the process

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 3112ea3..HEAD -- cmd/template/main.go`
> Plan 008 may have turned `serve`, `routes`, and `scrape` into methods; that
> does not affect this plan. The first lines of `run` must still match the
> excerpt below; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `3112ea3`, 2026-11-23

## Why this matters

The entrypoint turns `SIGINT` and `SIGTERM` into context cancellation with
`signal.NotifyContext` and releases the signal handler only when `run`
returns. While graceful shutdown is in progress the signals are still
captured, so pressing Ctrl-C a second time does nothing. If shutdown hangs —
a workload that ignores its context, or a database pool waiting for a
connection that is never released — the only way out is `SIGKILL`. Restoring
default signal handling as soon as the first signal arrives gives operators
the conventional escape hatch: the first signal asks politely, the second one
terminates.

## Current state

- `cmd/template/main.go:61-66`:

  ```go
  // run loads configuration, opens shared dependencies, and runs every
  // workload until all of them finish, one of them fails, or the process
  // receives an interrupt or termination signal.
  func run() error {
  	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
  	defer stop()
  ```

- Later in the same function (`main.go:84`, `main.go:105`) a deferred
  `database.Close()` runs after `group.Wait()` returns, with no timeout,
  while the signals are still captured.

- `cmd/template/main.go` contains setup markers: full-line comments such as
  `// [server]` and `// [/server]`. The lines above are outside every marker
  block; keep the new code outside them too, so it survives every scaffold
  shape.

- `go.mod:3` declares Go 1.27.2, so `context.AfterFunc` (added in Go 1.21) is
  available.

Conventions that apply (from the repository's `AGENTS.md` and `STYLE.md`):

- Every started goroutine and acquired resource has an explicit owner and
  outcome.
- Comments explain behavior that is not obvious from the code, briefly.

## Commands you will need

A local, git-ignored `go.work` may exist on the maintainer's machine. Prefix
Go commands with `GOWORK=off` so results match CI.

| Purpose      | Command                                                                                | Expected on success |
|--------------|----------------------------------------------------------------------------------------|---------------------|
| Build        | `GOWORK=off go build ./...`                                                            | exit 0, no output   |
| Vet          | `GOWORK=off go vet ./...`                                                              | exit 0, no output   |
| Format check | `gofmt -l cmd`                                                                         | no output           |
| Lint         | `GOWORK=off go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run` | exit 0, `0 issues.` |

Run lint once before changing anything to learn the baseline; run it in the
background.

## Scope

**In scope** (the only files you should modify):

- `cmd/template/main.go` (the top of `run` and its doc comment)

**Out of scope** (do NOT touch, even though they look related):

- `serve`, `shutdownServer`, `scrape`, and the errgroup wiring.
- Adding a timeout to `database.Close()`.
- `cmd/setup/main.go`, which has its own `signal.NotifyContext`.
- Every setup marker line.
- Any test file. This repository forbids adding or running tests unless the
  maintainer asks for them.
- Uncommitted maintainer work in `README.md`, `AGENTS.md`, `STYLE.md`,
  `pkg/service/service.go`, and `.agents/skills/go-style/`. Do not stash,
  revert, commit, or reformat those files.

## Git workflow

- Branch: `advisor/010-second-signal-ends-shutdown`
- One commit. Message style is a short imperative sentence, as in `git log`:
  "Use fphttp v2 profiles in the client". Suggested:
  `Restore default signal handling once shutdown begins`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Release the signal handler when the first signal arrives

In `run` in `cmd/template/main.go`, directly after `defer stop()`, register
`stop` to run when the context is done:

```go
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Once shutdown begins, a second signal terminates the process.
	context.AfterFunc(ctx, stop)
```

`stop` is safe to call more than once, so the existing `defer stop()` stays.
This must come before the line `group, ctx := errgroup.WithContext(ctx)`
later in the function, which shadows `ctx`; placing it right after
`defer stop()` guarantees that.

Extend the doc comment of `run` with one sentence saying that a second signal
during shutdown terminates the process immediately.

**Verify**: `grep -n "context.AfterFunc(ctx, stop)" cmd/template/main.go` →
one match, on a line before the `errgroup.WithContext` line.

### Step 2: Full verification

Run build, vet, format check, and lint from the commands table.

**Verify**: each command gives its expected result, and
`grep -c "// \[" cmd/template/main.go` prints the same number as
`git show HEAD:cmd/template/main.go | grep -c "// \["`.

## Test plan

No tests. This repository's rules forbid adding, running, or proposing tests
unless the maintainer explicitly asks. Verification is the build, vet,
format, and lint commands and a read of the changed lines.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `GOWORK=off go build ./...` and `GOWORK=off go vet ./...` exit 0
- [ ] `gofmt -l cmd` prints nothing
- [ ] Lint reports no issue that was not in the baseline
- [ ] `grep -c "context.AfterFunc(ctx, stop)" cmd/template/main.go` prints `1`
- [ ] The marker count in `cmd/template/main.go` is unchanged
- [ ] `git diff --stat` lists only `cmd/template/main.go`
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The top of `run` does not match the excerpt.
- Lint objects to the unused return value of `context.AfterFunc`. Report the
  exact message; do not add a `//nolint` comment.
- The fix appears to require touching an out-of-scope function.

## Maintenance notes

- The context is also done when the function returns normally; calling
  `stop` then is harmless.
- If a workload fails on its own, the errgroup cancels its derived context,
  not this one, so signals stay captured until `run` returns. That is fine:
  the first signal during that shutdown still cancels cleanly and releases
  the handler.
- Deferred: `database.Close()` has no deadline. With this change an operator
  can interrupt a hung close, which was judged sufficient.

Standards: Meet the solution standards in ~/.agents/AGENTS.md: idiomatic, beautiful, elegant, seamless, well-structured, dynamic, and future-proof, as explained in ~/.agents/CODING.md.
Guidance: Before implementing, read the repository's AGENTS.md and STYLE.md (if present in your checkout), load the go-style skill (`.agents/skills/go-style/SKILL.md`), and read `cmd/template/main.go` in full.
Review: Read every changed file from top to bottom for readability before declaring the plan done.
Evidence: Verify with build, vet, formatting checks, lint, command output, and the relevant file contents. Do not add or run tests unless tests are explicitly requested.
Execution: Run every shell command that may take more than a few seconds, and every subagent task, in the background, and keep working on what does not depend on it until its completion notification arrives. Never sleep or poll.
