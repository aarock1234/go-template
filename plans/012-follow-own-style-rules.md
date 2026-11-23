# Plan 012: The template's own code and style examples follow its style rules

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 3112ea3..HEAD -- cmd/setup .agents/skills/go-style/references`
> Plans 005–008 are expected to have changed `cmd/setup`. Part A below tells
> you how to handle that. For the skill files, compare the "Current state"
> excerpts against the live files; on a mismatch, treat it as a STOP
> condition.
>
> **Precondition (run second)**: `git status --short .agents` must print
> nothing. If it lists modified files, the maintainer has uncommitted edits
> to the skill: do Part A only, mark Part B as BLOCKED in your report, and
> ask the operator to commit those edits first.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW (layout and documentation only)
- **Depends on**: plans/005-setup-reports-its-real-outcome.md, plans/007-setup-prunes-the-readme.md, plans/008-clean-scaffold-without-postgres.md (same files; land them first)
- **Category**: tech-debt
- **Planned at**: commit `3112ea3`, 2026-11-23

## Why this matters

This repository exists to demonstrate a coding style, and two of its rules
are broken by the repository itself. The headline rule — expand every
nonempty struct literal, even with one field — is violated in `cmd/setup`,
and no formatter or linter catches that. And the bundled style guide's code
examples log with bare key-value pairs and no context, which the
repository's own linter configuration forbids. People and agents who copy
from the guide produce code the template's CI rejects. A style reference
that contradicts itself teaches the wrong thing.

## Current state

### The rules

From the repository's `AGENTS.md`:

> Expand every nonempty struct declaration, struct literal, and keyed map
> literal, including one-field options, nested arguments, and returns. Use
> keyed struct fields and preserve gofmt alignment.

From `STYLE.md`: "Empty values such as `Service{}` or `map[string]string{}`
may stay compact. Collections of simple values are a separate layout
decision." So `[]string{"pkg/worker"}` may stay on one line; the struct
literal around it may not.

From `.golangci.yml:12-17`, the logging rules the linter enforces:

```yaml
  settings:
    sloglint:
      attr-only: true        # slog.String("key", v), never bare key-value pairs
      context: scope         # use the *Context variants when a context is in scope
      key-naming-case: snake
      msg-style: lowercased
```

A conforming call in real code, `cmd/template/main.go:56`:

```go
		slog.Error("fatal error", slog.Any("error", err))
```

### Part A — `cmd/setup`

At the planned commit there are seven one-line struct literals:

- `cmd/setup/marker.go:105`:

  ```go
  		return markedLine{text: raw}
  ```

- `cmd/setup/feature.go:122`, and the same shape at lines 127, 132, 137,
  142, and 147:

  ```go
  		feature: feature{paths: []string{"pkg/worker"}},
  ```

Plan 007 removes the `feature:` field from those six entries. If it has
landed, only the `marker.go` literal remains.

### Part B — the bundled style guide

Files under `.agents/skills/go-style/references/`. Code examples in these
files are indented with four spaces, not tabs; keep that.

- `architecture.md:125-127`, inside an HTTP handler that has `r` in scope:

  ```go
      if err := json.NewEncoder(w).Encode(item); err != nil {
          h.logger.Error("encode response", "error", err)
      }
  ```

- `architecture.md:142-147`:

  ```go
  func main() {
      if err := run(); err != nil {
          slog.Error("fatal error", "error", err)
          os.Exit(1)
      }
  }
  ```

- `architecture.md:173`, inside `run()`, which has no context variable:

  ```go
      logger.Info("server starting", "port", "8080")
  ```

- `errors.md:39-44`, a deliberately bad example about logging and returning
  the same error, inside a method that has `ctx` in scope:

  ```go
  // BAD: logs and returns; caller will likely log again
  if err != nil {
      s.logger.Error("failed to get item", "error", err)
      return nil, fmt.Errorf("get item: %w", err)
  }
  ```

- `structs.md:55-60`, a struct whose last field is not aligned:

  ```go
  type Service struct {
      repo   Repository
      logger *slog.Logger
      timeout time.Duration
  }
  ```

## Commands you will need

A local, git-ignored `go.work` may exist on the maintainer's machine. Prefix
Go commands with `GOWORK=off` so results match CI.

| Purpose      | Command                                                                                | Expected on success |
|--------------|----------------------------------------------------------------------------------------|---------------------|
| Build        | `GOWORK=off go build ./...`                                                            | exit 0, no output   |
| Vet          | `GOWORK=off go vet ./...`                                                              | exit 0, no output   |
| Format check | `gofmt -l cmd`                                                                         | no output           |
| Lint         | `GOWORK=off go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run` | exit 0, `0 issues.` |

Run lint in the background.

## Scope

**In scope** (the only files you should modify):

- `cmd/setup/marker.go`
- `cmd/setup/feature.go` (only if the `feature: feature{...}` literals still
  exist)
- `.agents/skills/go-style/references/architecture.md`
- `.agents/skills/go-style/references/errors.md`
- `.agents/skills/go-style/references/structs.md`

**Out of scope** (do NOT touch, even though they look related):

- Any behavior. Part A is layout only; no identifier, value, or control flow
  changes.
- Slices of plain strings such as `[]string{"pkg/db", "pkg/service"}`; the
  style guide lets collections of simple values stay on one line.
- Other examples in the skill, the prose around the examples, `SKILL.md`,
  and `foundations.md` (which mentions a path in the maintainer's home
  directory; that is the maintainer's call).
- `pkg/db/sqlc/` — generated code is never hand-formatted.
- Any test file. This repository forbids adding or running tests unless the
  maintainer asks for them.
- Uncommitted maintainer work in `README.md`, `AGENTS.md`, `STYLE.md`, and
  `pkg/service/service.go`. Do not stash, revert, commit, or reformat it.

## Git workflow

- Branch: `advisor/012-follow-own-style-rules`
- Two commits, one per part. Message style is a short imperative sentence,
  as in `git log`: "Use fphttp v2 profiles in the client". Suggested:
  `Expand struct literals in setup` and
  `Make style guide examples pass the configured linter`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1 (Part A): Expand the remaining one-line struct literals in `cmd/setup`

Find them:

```bash
grep -n "markedLine{text: raw}\|feature: feature{" cmd/setup/*.go
```

Expand each match to one field per line. For example:

```go
		return markedLine{
			text: raw,
		}
```

and, only if these still exist:

```go
		feature: feature{
			paths: []string{"pkg/worker"},
		},
```

Then read every file in `cmd/setup/` once and expand any other nonempty
struct literal that sits on one line. Run `gofmt -w` on the files you
changed.

**Verify**: the grep above → no matches; `gofmt -l cmd` → no output;
`GOWORK=off go build ./...` → exit 0.

### Step 2 (Part B): Make the logging examples conform

Skip this step if the precondition at the top found uncommitted changes
under `.agents`.

Change exactly these four lines, keeping their indentation:

- `architecture.md`, the encode failure in the handler:
  `h.logger.ErrorContext(r.Context(), "encode response", slog.Any("error", err))`
- `architecture.md`, in `main`:
  `slog.Error("fatal error", slog.Any("error", err))`
- `architecture.md`, in `run`:
  `logger.Info("server starting", slog.String("port", "8080"))`
- `errors.md`, the BAD example (it stays a bad example; only its logging
  form changes):
  `s.logger.ErrorContext(ctx, "failed to get item", slog.Any("error", err))`

**Verify**:
`grep -rn '"error", err)' .agents/skills/go-style/references | grep -v 'slog.Any'`
→ no matches, and
`grep -n '"port", "8080"' .agents/skills/go-style/references/architecture.md`
→ one match, containing `slog.String`.

### Step 3 (Part B): Align the struct example

Skip this step if the precondition found uncommitted changes under `.agents`.

In `structs.md`, align the three fields of the `Service` example the way
gofmt would, keeping four-space indentation:

```go
type Service struct {
    repo    Repository
    logger  *slog.Logger
    timeout time.Duration
}
```

**Verify**: `grep -n "timeout time.Duration" .agents/skills/go-style/references/structs.md`
→ one match whose type column lines up with the two lines above it.

### Step 4: Full verification

Run build, vet, format check, and lint from the commands table.

**Verify**: each command gives its expected result, and `git diff --stat`
for Part B shows four changed lines in `architecture.md` and `errors.md`
combined and three in `structs.md`.

## Test plan

No tests. This repository's rules forbid adding, running, or proposing tests
unless the maintainer explicitly asks. Verification is the build, vet,
format, and lint commands and the grep checks.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `GOWORK=off go build ./...` and `GOWORK=off go vet ./...` exit 0
- [ ] `gofmt -l cmd` prints nothing
- [ ] Lint reports no issue that was not in the baseline
- [ ] `grep -n "markedLine{text: raw}\|feature: feature{" cmd/setup/*.go` returns no matches
- [ ] Part B done, or reported as BLOCKED because of uncommitted maintainer edits: `grep -rn '"error", err)' .agents/skills/go-style/references | grep -v 'slog.Any'` returns no matches
- [ ] `git diff --stat` lists only in-scope files
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- A skill excerpt above does not match the live file.
- Expanding a literal would change behavior or needs a `//nolint` comment.
- You find the skill directory is a copy that some tool synchronizes from
  another location (for example a note at the top of `SKILL.md` saying so).
  Editing a synchronized copy would be overwritten; report it instead.
- A verification fails twice after a reasonable fix attempt.

## Maintenance notes

- Nothing enforces the struct-literal rule mechanically. Reviewers have to
  look for it, as the repository's guidance says: "A formatter passing is
  not a readability review."
- The maintainer keeps a personal copy of the go-style skill outside this
  repository. If the two are meant to stay identical, the same four lines
  need the same edit there.
- Other examples in the skill were not audited line by line; only the
  logging calls that the configured linter would reject and the one
  misaligned struct were fixed.

Standards: Meet the solution standards in ~/.agents/AGENTS.md: idiomatic, beautiful, elegant, seamless, well-structured, dynamic, and future-proof, as explained in ~/.agents/CODING.md.
Guidance: Before implementing, read the repository's AGENTS.md and STYLE.md (if present in your checkout), load the go-style skill (`.agents/skills/go-style/SKILL.md`), and read its `references/structs.md` and `references/logging.md`.
Review: Read every changed file from top to bottom for readability before declaring the plan done.
Evidence: Verify with build, vet, formatting checks, lint, command output, and the relevant file contents. Do not add or run tests unless tests are explicitly requested.
Execution: Run every shell command that may take more than a few seconds, and every subagent task, in the background, and keep working on what does not depend on it until its completion notification arrives. Never sleep or poll.
