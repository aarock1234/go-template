# Plan 009: The application is the only thing that parses `.env` for itself

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 3112ea3..HEAD -- Makefile .env.example`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition. (Plan 004 adds comment lines to
> `.env.example`; that is expected.)

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: MED (`make migrate` gets its database URL a different way)
- **Depends on**: none
- **Category**: dx
- **Planned at**: commit `3112ea3`, 2026-11-23

## Why this matters

The Makefile begins with `-include .env` and a bare `export`. Make therefore
parses `.env` with Makefile syntax and exports every value to every recipe.
The application has its own `.env` loader in `pkg/env`, with different rules,
and it lets the process environment win. So under `make dev` the value make
parsed overrides the value the application would have parsed. The two
disagree whenever a value is quoted (make keeps the quotes), contains `#`
(make starts a comment), or contains `$` (make expands it). A database
password with any of those works with `go run ./cmd/template` and fails with
`make dev`. Every secret in `.env` is also exported to tools that do not need
it, such as the linter. After this plan make no longer reads `.env` globally;
only the migration targets load it, through the shell, and the one remaining
difference is documented.

## Current state

- `Makefile:1-10`:

  ```make
  -include .env
  export

  # Tool versions. Keep GOLANGCI_LINT_VERSION in sync with .github/workflows/ci.yaml.
  GOLANGCI_LINT_VERSION := v2.14.0
  GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
  # [postgres]
  GOOSE_VERSION := v3.29.0
  GOOSE := go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir pkg/db/migrations
  # [/postgres]
  ```

- `Makefile:70-82` — the only recipes that use a value from `.env`:

  ```make
  # [postgres]
  .PHONY: migrate migrate-down migrate-new

  migrate: ## Apply all migrations
  	$(GOOSE) postgres "$(DATABASE_URL)" up

  migrate-down: ## Roll back the last migration
  	$(GOOSE) postgres "$(DATABASE_URL)" down

  migrate-new: ## Create a migration (make migrate-new name=create_users)
  	@test -n "$(name)" || { echo "usage: make migrate-new name=<migration_name>"; exit 1; }
  	$(GOOSE) create $(name) sql
  # [/postgres]
  ```

- Everything else gets its configuration without make's help:
  - `make dev` runs `go run ./cmd/template`, which loads `.env` through
    `pkg/env`.
  - `make up`, `make watch`, `make db` run Docker Compose, which reads `.env`
    from the project directory itself (`compose.yaml:6-8` and the
    `${PORT:-8080}` substitution).

- The application's rules, `pkg/env/env.go:23-30`:

  ```go
  // Load reads KEY=value lines from a .env file and sets them in the process
  // environment. Existing variables are not overwritten, a missing file is
  // not an error, and repeated calls with the same path return the cached
  // result.
  //
  // Blank lines and lines starting with # are ignored, an optional "export "
  // prefix is allowed, and values may be wrapped in matching single or double
  // quotes. If no path is provided, ".env" is used.
  ```

- `.env.example:1-2` starts with:

  ```
  # Log level: debug, info, warn, error
  LOG_LEVEL=debug
  ```

- The `# [tag]` and `# [/tag]` lines are setup markers used by the one-shot
  tool in `cmd/setup`. Never add text to a marker line, and keep every block
  balanced. Anything specific to PostgreSQL must sit inside a `[postgres]`
  block so it is removed with that feature.

- The Makefile already assumes a POSIX shell (`test -n`, `{ ...; }`, `awk`).
  Recipe lines are indented with a tab, not spaces.

## Commands you will need

| Purpose             | Command                         | Expected on success                                   |
|---------------------|---------------------------------|-------------------------------------------------------|
| List targets        | `make help`                     | exit 0, prints the target list                        |
| Preview a recipe    | `make -n migrate`               | exit 0, prints the commands without running them      |
| Preview dev         | `make -n dev`                   | prints `go run ./cmd/template`                        |

`make -n` does not execute recipes, so these are safe without a database.
If `make` is not installed, verify by reading the file and say so in your
report.

## Scope

**In scope** (the only files you should modify):

- `Makefile`
- `.env.example` (one short comment at the top)

**Out of scope** (do NOT touch, even though they look related):

- `pkg/env/` — the application's parser does not change. In particular, do
  not add inline-comment support.
- `compose.yaml`, `README.md`, `cmd/`.
- Every other Makefile target and every setup marker line.
- Any test file. This repository forbids adding or running tests unless the
  maintainer asks for them.
- Uncommitted maintainer work in `README.md`, `AGENTS.md`, `STYLE.md`,
  `pkg/service/service.go`, and `.agents/skills/go-style/`. Do not stash,
  revert, commit, or reformat those files.

## Git workflow

- Branch: `advisor/009-one-env-parser`
- One commit. Message style is a short imperative sentence, as in `git log`:
  "Use fphttp v2 profiles in the client". Suggested:
  `Stop parsing .env with make`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Remove the global include and export

Delete the first two lines of the `Makefile` (`-include .env` and `export`)
and the blank line that follows them, so the file starts with the
"Tool versions" comment.

**Verify**: `grep -n "^-include\|^export" Makefile` → no matches.

### Step 2: Load the database URL for the migration targets only

Inside the first `[postgres]` block (next to `GOOSE_VERSION` and `GOOSE`),
define a shell fragment that makes `DATABASE_URL` available to a recipe: it
keeps a value already in the process environment, and otherwise loads `.env`
with the shell. Target shape:

```make
# Takes DATABASE_URL from the environment, or from .env when it is unset
# there, as the application does.
LOAD_DATABASE_URL := [ -n "$$DATABASE_URL" ] || { set -a; . ./.env; set +a; };
```

Then change the two recipes that need it so the shell, not make, expands the
variable:

```make
migrate: ## Apply all migrations
	@$(LOAD_DATABASE_URL) $(GOOSE) postgres "$$DATABASE_URL" up

migrate-down: ## Roll back the last migration
	@$(LOAD_DATABASE_URL) $(GOOSE) postgres "$$DATABASE_URL" down
```

Notes:

- `$$` is how a Makefile writes a literal `$` for the shell.
- The leading `@` keeps make from echoing the line, as `migrate-new` already
  does for its guard.
- Leave `migrate-new` unchanged; it does not use the database URL.

**Verify**:

- `grep -n '\$(DATABASE_URL)' Makefile` → no matches.
- `make -n migrate` → exit 0; the printed command contains `. ./.env` and
  `"$DATABASE_URL" up`.
- `make -n dev` → prints `go run ./cmd/template`.
- `make help` → exit 0 and still lists `migrate`, `migrate-down`,
  `migrate-new`.
- `grep -c "^# \[" Makefile` prints the same number as
  `git show HEAD:Makefile | grep -c "^# \["`.

### Step 3: Document the one rule that keeps both readers in agreement

The application and the shell now both read `.env` (the shell only for the
migration targets). They agree on plain values and on single-quoted values.
At the very top of `.env.example`, before the `LOG_LEVEL` comment, add a
two- or three-line comment saying: values are read literally; wrap a value in
single quotes if it contains spaces, `#`, or `$`. Follow it with a blank
line. Do not put this comment inside any marker block.

**Verify**: `head -5 .env.example` shows the new comment followed by a blank
line, and `grep -c "^# \[" .env.example` prints the same number as
`git show HEAD:.env.example | grep -c "^# \["`.

## Test plan

No tests. This repository's rules forbid adding, running, or proposing tests
unless the maintainer explicitly asks. Verification is the `make -n` previews
and the grep checks. If a PostgreSQL instance and a `.env` are available, a
real `make migrate` is useful extra evidence, but it is not required and must
be run in the background.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -n "^-include\|^export" Makefile` returns no matches
- [ ] `grep -n '\$(DATABASE_URL)' Makefile` returns no matches
- [ ] `grep -c "LOAD_DATABASE_URL" Makefile` prints `3`
- [ ] `make -n migrate` exits 0 and its output contains `. ./.env`
- [ ] `make help` exits 0
- [ ] Marker counts in `Makefile` and `.env.example` are unchanged
- [ ] `git diff --stat` lists only `Makefile` and `.env.example`
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The Makefile's first lines or the migration recipes do not match the
  excerpts.
- Any recipe other than `migrate` and `migrate-down` turns out to reference
  a variable that only `.env` defines (search for `$(` followed by an
  upper-case name that is not assigned in the Makefile).
- `make -n migrate` fails or prints a command in which `DATABASE_URL` was
  expanded by make.
- The fix appears to require changing `pkg/env`.
- A verification fails twice after a reasonable fix attempt.

## Maintenance notes

- Behavior change: `make dev`, `make lint`, and the rest no longer receive
  `.env` values from make. Anything that relied on that must read the
  environment itself.
- Two readers of `.env` remain (the application and, for migrations, the
  shell). They differ on double-quoted values containing `$` and on unquoted
  values containing spaces; the comment in `.env.example` tells users how to
  avoid both. A single reader would need goose to accept `DATABASE_URL`
  directly or a small Go migration command; both were judged larger than the
  problem.
- If `.env` is missing and `DATABASE_URL` is unset, `make migrate` now fails
  with the shell's "No such file" message. Reviewers may want a friendlier
  message; it was left out to keep the fragment short.

Standards: Meet the solution standards in ~/.agents/AGENTS.md: idiomatic, beautiful, elegant, seamless, well-structured, dynamic, and future-proof, as explained in ~/.agents/CODING.md.
Guidance: Before implementing, read the repository's AGENTS.md and STYLE.md (if present in your checkout), and read `Makefile`, `.env.example`, and `pkg/env/env.go` in full.
Review: Read every changed file from top to bottom for readability before declaring the plan done.
Evidence: Verify with command output and the relevant file contents. Do not add or run tests unless tests are explicitly requested.
Execution: Run every shell command that may take more than a few seconds, and every subagent task, in the background, and keep working on what does not depend on it until its completion notification arrives. Never sleep or poll.
