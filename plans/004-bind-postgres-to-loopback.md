# Plan 004: The bundled PostgreSQL is reachable from the local machine only

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 3112ea3..HEAD -- compose.yaml .env.example`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none (touches `compose.yaml`, as plan 002 does, in a different block)
- **Category**: security
- **Planned at**: commit `3112ea3`, 2026-11-23

## Why this matters

`make db` and `make up` start a PostgreSQL container whose superuser name and
password are fixed, well-known development defaults. The compose file
publishes its port without a host address, which Docker binds on every
network interface. The Makefile and README both describe the database as
being on `localhost:5432`, so on a laptop on shared Wi-Fi or on a cloud VM
the database is exposed to the network contrary to what the documentation
says. Binding the published port to the loopback address makes the behavior
match the documentation and makes the default credentials harmless.

## Current state

- `compose.yaml:32-48` — the database service:

  ```yaml
    # [postgres-docker]
    postgres:
      image: postgres:18-alpine
      environment:
        POSTGRES_USER: postgres
        POSTGRES_PASSWORD: postgres
        POSTGRES_DB: app
      ports:
        - "5432:5432"
      volumes:
        - postgres-data:/var/lib/postgresql
      healthcheck:
        test: ["CMD-SHELL", "pg_isready -U postgres -d app"]
        interval: 5s
        timeout: 5s
        retries: 5
    # [/postgres-docker]
  ```

- `.env.example:9-16`:

  ```
  # [postgres]
  # PostgreSQL connection string
  # [postgres-docker]
  # `make db` starts a local instance on localhost:5432. Docker Compose points
  # the app container at its bundled database automatically.
  # [/postgres-docker]
  DATABASE_URL=postgres://postgres:postgres@localhost:5432/app?sslmode=disable
  # [/postgres]
  ```

- `Makefile:63` documents the target as "Start only PostgreSQL on
  localhost:5432".

The `# [tag]` and `# [/tag]` lines are setup markers. The one-shot tool in
`cmd/setup` removes the marked blocks for features a user disables and strips
the marker lines themselves. Never add text to a marker line, and keep every
marker balanced. A new comment that only makes sense when the bundled
database exists must sit inside the `[postgres-docker]` block.

## Commands you will need

| Purpose          | Command                                   | Expected on success               |
|------------------|-------------------------------------------|-----------------------------------|
| Binding present  | `grep -n '127.0.0.1:5432:5432' compose.yaml` | one match                      |
| Compose is valid | `docker compose config --quiet`           | exit 0, no output (needs Docker)  |

`docker compose config` is optional evidence; run it only if Docker is
installed.

## Scope

**In scope** (the only files you should modify):

- `compose.yaml` (the `ports` entry of the `postgres` service)
- `.env.example` (the comment inside the `[postgres-docker]` block)

**Out of scope** (do NOT touch, even though they look related):

- The `app` service's published port (`compose.yaml:10-11`). A development
  HTTP server reachable from other devices is a deliberate convenience.
- The default credentials themselves and `sslmode=disable`. Replacing them
  with required variables would make `make db` fail on a fresh checkout; with
  the port on loopback the defaults are acceptable for local development.
- `Makefile`, `README.md`, `Dockerfile`.
- Every setup marker line.
- Uncommitted maintainer work in `README.md`, `AGENTS.md`, `STYLE.md`,
  `pkg/service/service.go`, and `.agents/skills/go-style/`. Do not stash,
  revert, commit, or reformat those files.

## Git workflow

- Branch: `advisor/004-bind-postgres-to-loopback`
- One commit. Message style is a short imperative sentence, as in `git log`:
  "Use fphttp v2 profiles in the client". Suggested:
  `Publish the bundled PostgreSQL on loopback only`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Bind the published port to loopback

In `compose.yaml`, change the `postgres` service's port entry from
`"5432:5432"` to `"127.0.0.1:5432:5432"`. Change nothing else in the file.

**Verify**: `grep -n '127.0.0.1:5432:5432' compose.yaml` → one match, and
`git diff compose.yaml` shows exactly one changed line.

### Step 2: Say that the defaults are for local development

In `.env.example`, extend the comment inside the `[postgres-docker]` block
(between the `# [postgres-docker]` and `# [/postgres-docker]` lines) with one
more comment line stating that the user, password, and `sslmode=disable` in
the default URL are local development defaults for that bundled instance and
must be replaced for any other database. Keep it to one or two lines, wrapped
like the existing comment.

**Verify**: `grep -c "^# \[" .env.example` → prints `8`, the same count as
before your change (markers untouched), and `git diff .env.example` shows
only added comment lines inside the `[postgres-docker]` block.

### Step 3: Optional Docker evidence

If Docker is available, run `docker compose config --quiet`.

**Verify**: exit 0. If Docker is not available, say so in your report.

## Test plan

No tests. This repository's rules forbid adding, running, or proposing tests
unless the maintainer explicitly asks. Verification is the grep checks, the
diff inspection, and the optional compose validation.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -c '127.0.0.1:5432:5432' compose.yaml` prints `1`
- [ ] `grep -n '"5432:5432"' compose.yaml` returns no matches
- [ ] `grep -c "^# \[" .env.example` prints `8`
- [ ] `git diff --stat` lists only `compose.yaml` and `.env.example`
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The `postgres` service or the `.env.example` block does not match the
  excerpts above.
- The marker count in `.env.example` is not `8` before you start.
- `docker compose config` rejects the new port syntax.
- The fix appears to require touching an out-of-scope file.

## Maintenance notes

- Anyone who needs the development database from another machine must now
  opt in by editing the binding. That is the intent.
- The default credentials still appear in `compose.yaml` and `.env.example`.
  If the template ever grows a production compose file, those must become
  required variables there.
- If these defaults were ever used for a database reachable from a network,
  rotate that database's password.

Standards: Meet the solution standards in ~/.agents/AGENTS.md: idiomatic, beautiful, elegant, seamless, well-structured, dynamic, and future-proof, as explained in ~/.agents/CODING.md.
Guidance: Before implementing, read the repository's AGENTS.md and STYLE.md (if present in your checkout), and read `compose.yaml` and `.env.example` in full.
Review: Read every changed file from top to bottom for readability before declaring the plan done.
Evidence: Verify with command output and the relevant file contents. Do not add or run tests unless tests are explicitly requested.
Execution: Run every shell command that may take more than a few seconds, and every subagent task, in the background, and keep working on what does not depend on it until its completion notification arrives. Never sleep or poll.
