# Plan 008: A server-only scaffold without PostgreSQL passes lint and has no orphaned handler

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 3112ea3..HEAD -- cmd/template/main.go cmd/setup/feature.go pkg/handler`
> Plans 005–007 are expected to have changed `cmd/setup/feature.go`. The
> `postgresRemoval` spec and `cmd/template/main.go` must still match the
> excerpts below; on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: plans/005-setup-reports-its-real-outcome.md (same file; land it first)
- **Category**: bug
- **Planned at**: commit `3112ea3`, 2026-11-23

## Why this matters

The one-shot setup tool lets a user keep the HTTP server and drop PostgreSQL
(`-postgres=false`). That scaffold compiles, but it has two leftovers. The
`routes` function keeps an `app` parameter whose only uses were removed, and
the repository's linter (revive, with its default `unused-parameter` rule)
is expected to reject that, so the user's first CI run fails on code they
did not write. And `pkg/handler/handler.go` stays behind: an example handler
for a service package that setup has just deleted. A template's first
impression is its scaffold; it should come out lint-clean and free of dead
example code.

## Current state

- `cmd/template/main.go` — the entrypoint. Feature-specific code is wrapped
  in setup markers: a block runs from a full-line `// [tag]` comment to the
  matching `// [/tag]`. Setup removes the blocks of disabled features and
  strips all marker comments.

  The dependency holder, `cmd/template/main.go:46-52`:

  ```go
  // app holds the dependencies shared by the process's workloads.
  type app struct {
  	config *Config
  	// [postgres]
  	database *db.DB
  	// [/postgres]
  }
  ```

  The workloads are started in `run`, `cmd/template/main.go:93-103`:

  ```go
  	// [server]
  	group.Go(func() error {
  		return serve(ctx, app)
  	})
  	// [/server]

  	// [client]
  	group.Go(func() error {
  		return scrape(ctx, app)
  	})
  	// [/client]
  ```

  `serve` (`main.go:113`) is `func serve(ctx context.Context, app *app) error`
  and builds the server with `Handler: routes(app),` (`main.go:116`).
  `scrape` (`main.go:215`) is
  `func scrape(ctx context.Context, app *app) error`.

  The routes, `cmd/template/main.go:156-177`:

  ```go
  // routes builds the HTTP handler tree. Middleware order matters: the
  // request ID is assigned first so the logger and panic recovery can use it,
  // and recovery runs inside the logger so recovered panics are logged as 500s.
  func routes(app *app) http.Handler {
  	router := chi.NewRouter()
  	router.Use(middleware.RequestID)
  	router.Use(middleware.Logger)
  	router.Use(middleware.Recover)

  	router.Get("/livez", handler.Live)
  	// [postgres]
  	router.Get("/readyz", handler.Ready(app.database))

  	example := handler.New(service.New(app.database))

  	router.Route("/api", func(r chi.Router) {
  		r.Get("/example", example.GetExample)
  	})
  	// [/postgres]

  	return router
  }
  ```

  With the `[postgres]` block removed, `app` is never used inside `routes`.

- `.golangci.yml:10` enables `revive` with its default rules. The repository
  already avoids the rule elsewhere: `pkg/handler/health.go:22` declares
  `func Live(w http.ResponseWriter, _ *http.Request)`.

- `pkg/handler/handler.go` — the example: `Service` interface, `Handler`,
  `New`, `GetExample`, and `writeError` (maps `apperr` errors to HTTP
  responses; used only by `GetExample`).
- `pkg/handler/health.go` — `Live`, `Ready`, `Pinger`. Generic; stays.

- `cmd/setup/feature.go:90-93` — what `-postgres=false` removes:

  ```go
  	postgresRemoval = feature{
  		paths: []string{"pkg/db", "pkg/service"},
  		tags:  []tag{tagPostgres, tagPostgresDocker},
  	}
  ```

  `pkg/service` is deleted, but `pkg/handler/handler.go`, whose `Handler`
  exists only to call that service, is not.

Conventions that apply (from the repository's `AGENTS.md` and `STYLE.md`):

- Every nonempty struct literal is multiline with keyed fields.
- Preserve setup markers exactly: never add text to a marker line, keep every
  block balanced.
- Doc comments state each contract; keep them accurate when a function
  becomes a method.

## Commands you will need

A local, git-ignored `go.work` may exist on the maintainer's machine. Prefix
Go commands with `GOWORK=off` so results match CI.

| Purpose      | Command                                                                                | Expected on success |
|--------------|----------------------------------------------------------------------------------------|---------------------|
| Build        | `GOWORK=off go build ./...`                                                            | exit 0, no output   |
| Vet          | `GOWORK=off go vet ./...`                                                              | exit 0, no output   |
| Format check | `gofmt -l cmd pkg`                                                                     | no output           |
| Lint         | `GOWORK=off go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run` | exit 0, `0 issues.` |

Run lint in the background. **Never run `go run ./cmd/setup` or `make setup`
in your working checkout**; the tool deletes files, including itself.
Behavior checks run only in a disposable worktree.

## Scope

**In scope** (the only files you should modify):

- `cmd/template/main.go`
- `cmd/setup/feature.go` (the `postgresRemoval` paths only)
- `pkg/handler/handler.go` and `pkg/handler/health.go` (only to move the
  package doc comment; see the amendment below)

> **Amendment (found during execution)**: the package comment
> `// Package handler provides HTTP handlers for the application.` lives in
> `pkg/handler/handler.go`, which Step 2 makes setup delete when PostgreSQL
> is disabled. The surviving `health.go` then has no package comment and
> revive's `package-comments` rule fails. Move that one line from the top of
> `handler.go` to directly above `package handler` in `health.go`, the file
> every scaffold keeps. Change nothing else in either file.

**Out of scope** (do NOT touch, even though they look related):

- Anything else in `pkg/handler/handler.go` and `pkg/handler/health.go`.
- `pkg/apperr` — stays in a server-only scaffold as a library package even
  though nothing imports it there.
- `.golangci.yml` — do not disable or configure the lint rule.
- Any marker line in `cmd/template/main.go`.
- Any test file. This repository forbids adding or running tests unless the
  maintainer asks for them.
- Uncommitted maintainer work in `README.md`, `AGENTS.md`, `STYLE.md`,
  `pkg/service/service.go`, and `.agents/skills/go-style/`. Do not stash,
  revert, commit, or reformat those files.

## Git workflow

- Branch: `advisor/008-clean-scaffold-without-postgres`
- One commit per step is fine. Message style is a short imperative sentence,
  as in `git log`: "Use fphttp v2 profiles in the client". Example:
  `Remove the example handler when PostgreSQL is disabled`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Reproduce the lint failure in a disposable worktree

```bash
CHECK="$(mktemp -d)/setup-check"
git worktree add --detach "$CHECK" HEAD
```

In `$CHECK`, run in the background: `go run ./cmd/setup -y -postgres=false`.
When it finishes, run the lint command from the commands table there (without
`GOWORK=off` if no `go.work` exists in the worktree), also in the background.

Record the result:

- **Expected**: lint reports an `unused-parameter` issue for `app` in
  `cmd/template/main.go`. Continue with all steps.
- **If lint reports no such issue**: skip Step 3, and say so in your report.
  Step 2 still applies.

Reset the worktree for later reuse:
`git -C "$CHECK" reset --hard && git -C "$CHECK" clean -fd`.

**Verify**: you have the lint output saved and know which branch applies.

### Step 2: Remove the example handler together with its service

In `cmd/setup/feature.go`, add `"pkg/handler/handler.go"` to the `paths` of
`postgresRemoval`. If the slice no longer reads comfortably on one line, put
one path per line.

**Verify**: `GOWORK=off go build ./cmd/setup` → exit 0, and
`grep -n "pkg/handler/handler.go" cmd/setup/feature.go` → one match.

### Step 3: Make the workloads methods of `app`

A method receiver is not a parameter, so it is not subject to the
unused-parameter rule, and "the app serves, scrapes, and builds its routes"
reads naturally. Convert all three functions so the entrypoint stays
consistent, in `cmd/template/main.go`:

- `func serve(ctx context.Context, app *app) error`
  → `func (a *app) serve(ctx context.Context) error`
- `func routes(app *app) http.Handler`
  → `func (a *app) routes() http.Handler`
- `func scrape(ctx context.Context, app *app) error`
  → `func (a *app) scrape(ctx context.Context) error`

Inside each body, replace `app.` with `a.`. Update the callers:

- in `run`: `return app.serve(ctx)` and `return app.scrape(ctx)`;
- in `serve`: `Handler: a.routes(),`.

Do not move, add, or remove any `// [tag]` or `// [/tag]` line, and keep each
function inside the marker block it is in today. Keep the doc comments;
adjust wording only where it names a parameter that no longer exists.

**Verify**: `GOWORK=off go build ./...` → exit 0;
`grep -n "func serve\|func routes\|func scrape" cmd/template/main.go` → no
matches; `grep -c "// \[" cmd/template/main.go` prints the same number as
`git show HEAD:cmd/template/main.go | grep -c "// \["`.

### Step 4: Static verification

Run build, vet, format check, and lint from the commands table. Commit.

**Verify**: each command gives its expected result.

### Step 5: Confirm every scaffold shape in the disposable worktree

Update the worktree to your new commit:
`git -C "$CHECK" checkout --detach <your commit SHA>`.

For each flag set below: run `go run ./cmd/setup <flags>` in `$CHECK` in the
background, then `go build ./...`, then the lint command, then reset with
`git -C "$CHECK" reset --hard && git -C "$CHECK" clean -fd`.

1. `-y -postgres=false` → setup exits 0; build exits 0; lint reports
   `0 issues.`; `test ! -e pkg/handler/handler.go` exits 0;
   `test -e pkg/handler/health.go` exits 0.
2. `-y` → setup exits 0; build exits 0; lint reports `0 issues.`;
   `test -e pkg/handler/handler.go` exits 0.
3. `-y -server=false` → setup exits 0; build exits 0; lint reports
   `0 issues.`.
4. `-y -client=false -postgres=false` → setup exits 0; build exits 0; lint
   reports `0 issues.`.

Remove the worktree:

```bash
git worktree remove --force "$CHECK"
```

**Verify**: all four shapes behave as stated, and `git worktree list` no
longer shows the throwaway checkout.

## Test plan

No tests. This repository's rules forbid adding, running, or proposing tests
unless the maintainer explicitly asks. Verification is the static commands
and the four scaffold shapes built and linted in a disposable worktree.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `GOWORK=off go build ./...` and `GOWORK=off go vet ./...` exit 0
- [ ] `gofmt -l cmd pkg` prints nothing
- [ ] Lint reports no issue that was not in the baseline
- [ ] `grep -n "pkg/handler/handler.go" cmd/setup/feature.go` matches once
- [ ] The marker count in `cmd/template/main.go` is unchanged
- [ ] All four scaffold shapes in Step 5 built and linted clean
- [ ] `git worktree list` shows no leftover throwaway worktree
- [ ] `git diff --stat` for your commits lists only `cmd/template/main.go`, `cmd/setup/feature.go`, and `plans/README.md`
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- `cmd/template/main.go` or `postgresRemoval` does not match the excerpts.
- Lint at the unmodified commit already fails in the full scaffold
  (`-y`) for reasons unrelated to this plan. Report the issues.
- Any scaffold shape in Step 5 fails lint for a reason other than the two
  this plan fixes. Report the shape and the issue; do not fix it here.
- Making a shape lint-clean appears to require changing `.golangci.yml` or
  adding a `//nolint` comment.
- You are about to run the setup tool anywhere other than the disposable
  worktree.
- A verification fails twice after a reasonable fix attempt.

## Maintenance notes

- `pkg/handler/handler.go` is now tied to PostgreSQL in the removal spec. If
  the example handler ever stops depending on the database-backed service,
  remove it from `postgresRemoval`.
- New code added to `cmd/template/main.go` inside a marker block should be
  checked against each scaffold shape; Step 5 is the procedure.
- Reviewers should confirm that no marker line moved in
  `cmd/template/main.go`.

Standards: Meet the solution standards in ~/.agents/AGENTS.md: idiomatic, beautiful, elegant, seamless, well-structured, dynamic, and future-proof, as explained in ~/.agents/CODING.md.
Guidance: Before implementing, read the repository's AGENTS.md and STYLE.md (if present in your checkout), load the go-style skill (`.agents/skills/go-style/SKILL.md`), and read `cmd/template/main.go` and `cmd/setup/feature.go` in full.
Review: Read every changed file from top to bottom for readability before declaring the plan done.
Evidence: Verify with build, vet, formatting checks, lint, command output, and the relevant file contents. Do not add or run tests unless tests are explicitly requested.
Execution: Run every shell command that may take more than a few seconds, and every subagent task, in the background, and keep working on what does not depend on it until its completion notification arrives. Never sleep or poll.
