# Plan 006: Setup points the import-grouping rule at the scaffolded module

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 3112ea3..HEAD -- cmd/setup .golangci.yml`
> Plan 005 is expected to have changed `cmd/setup`. Confirm that its changes
> are present (see "Current state") and that nothing else drifted; on any
> other mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: plans/005-setup-reports-its-real-outcome.md
- **Category**: dx
- **Planned at**: commit `3112ea3`, 2026-11-23

## Why this matters

Users create a project from this template with `gonew`, which rewrites the
module path in `go.mod` and the import paths in `.go` files. It does not
touch other files. `.golangci.yml` hardcodes the template's own module path
as the goimports "local prefix", the setting that makes the formatter put the
project's own imports in a separate group. In every scaffolded project that
prefix no longer matches anything, so `make format` silently stops enforcing
the import grouping the style guide requires. The one-shot setup tool already
finalizes the scaffold; it should fix this literal too.

## Current state

- `.golangci.yml:19-26`:

  ```yaml
  formatters:
    enable:
      - gofmt      # checks code formatting per gofmt
      - goimports  # checks import grouping and ordering
    settings:
      goimports:
        local-prefixes:
          - github.com/aarock1234/go-template
  ```

- `go.mod:1` — `module github.com/aarock1234/go-template`. In a scaffolded
  project this line holds the user's module path.

- `cmd/setup/` — the one-shot setup tool. `main.go` holds `apply`, which
  orchestrates the run; `feature.go` holds file-name constants and removal
  specs; `fs.go` holds file I/O helpers.

  File-name constants, `cmd/setup/feature.go:51-60`:

  ```go
  // Project files referenced by removal specs. Using constants turns a
  // misspelled name into a compile error.
  const (
  	fileMain         = "cmd/template/main.go"
  	fileMakefile     = "Makefile"
  	fileEnvExample   = ".env.example"
  	fileCompose      = "compose.yaml"
  	fileDockerfile   = "Dockerfile"
  	fileDockerignore = ".dockerignore"
  )
  ```

  After plan 005, the middle of `apply` in `cmd/setup/main.go` joins the file
  changes into `changeErr`:

  ```go
  	changeErr := errors.Join(
  		rewriteFiles(files, removal.tags),
  		removePaths(removal.paths),
  	)
  ```

  and then runs `go mod tidy` and `go fmt ./...`, returning an error if
  either the changes or the tidying failed. If `apply` still calls a function
  named `warn`, plan 005 has not landed: STOP.

- `gonew` copies string literals in `.go` files unchanged; it rewrites only
  import paths and `go.mod`. A Go constant holding the template's module path
  therefore still holds the template's path after scaffolding, which is what
  lets setup find the literal to replace.

Conventions that apply (from the repository's `AGENTS.md` and `STYLE.md`):

- Every nonempty struct literal is multiline with keyed fields.
- Separate I/O from decisions; small focused functions with doc comments.
- Error strings are lowercase, wrapped with `fmt.Errorf("context: %w", err)`.
- Files in `cmd/setup` are split by responsibility (`marker.go`, `fs.go`,
  `form.go`, `feature.go`). A new responsibility gets its own file.

## Commands you will need

A local, git-ignored `go.work` may exist on the maintainer's machine. Prefix
Go commands with `GOWORK=off` so results match CI.

| Purpose      | Command                                                                                | Expected on success |
|--------------|----------------------------------------------------------------------------------------|---------------------|
| Build        | `GOWORK=off go build ./...`                                                            | exit 0, no output   |
| Vet          | `GOWORK=off go vet ./...`                                                              | exit 0, no output   |
| Format check | `gofmt -l cmd`                                                                         | no output           |
| Lint         | `GOWORK=off go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run` | exit 0, `0 issues.` |

Run lint in the background. **Never run `go run ./cmd/setup` or `make setup`
in your working checkout**; the tool deletes files, including itself.
Behavior checks run only in a disposable worktree (Step 4).

## Scope

**In scope** (the only files you should modify):

- `cmd/setup/module.go` (create)
- `cmd/setup/main.go` (`apply` and its doc comment, and one sentence in the
  package comment)
- `cmd/setup/feature.go` (one new file-name constant)

**Out of scope** (do NOT touch, even though they look related):

- `.golangci.yml` — it must keep the template's own path so the template
  repository itself lints correctly.
- `README.md` — its title, badge, and `gonew` command also name the template
  repository. Plan 007 handles the README.
- Renaming `cmd/template` or the `template` binary. That is a larger change
  touching the Makefile, Dockerfile, and README.
- Any test file. This repository forbids adding or running tests unless the
  maintainer asks for them.
- Uncommitted maintainer work in `README.md`, `AGENTS.md`, `STYLE.md`,
  `pkg/service/service.go`, and `.agents/skills/go-style/`. Do not stash,
  revert, commit, or reformat those files.

## Git workflow

- Branch: `advisor/006-setup-rewrites-module-identity`
- One commit. Message style is a short imperative sentence, as in `git log`:
  "Use fphttp v2 profiles in the client". Suggested:
  `Point goimports at the scaffolded module during setup`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add the module-identity capability

Create `cmd/setup/module.go` in `package main` with:

- A constant for the template's module path,
  `github.com/aarock1234/go-template`, with a comment explaining that `gonew`
  leaves string literals alone, so this still names the template after
  scaffolding.
- A function that reads the module path from `go.mod` in the working
  directory: scan for the first line that starts with `module ` and return
  the rest, trimmed. Return an error if the file cannot be read or has no
  module line.
- A function that replaces every occurrence of one module path with another
  in a list of files, preserving everything else byte for byte, and joining
  individual failures with `errors.Join` as `rewriteFiles` in `fs.go` does.
- A small orchestrating function, called from `apply`, that reads the module
  path and, when it differs from the template's, rewrites the files. When the
  paths are equal it does nothing, so running setup inside the template
  repository itself changes nothing.

In `cmd/setup/feature.go`, add `fileGolangci = ".golangci.yml"` to the
file-name constant block, and declare the list of files that carry the module
path next to the capability in `module.go` (today it has one entry,
`fileGolangci`).

Give every function a doc comment that states its contract.

**Verify**: `GOWORK=off go build ./cmd/setup` → exit 0.

### Step 2: Call it from `apply`

In `cmd/setup/main.go`, add the new orchestrating function's result to the
`errors.Join` that produces `changeErr`, after `removePaths`. Update the
`apply` doc comment to mention that it also points tool configuration at the
project's module path. Add one sentence to the package comment at the top of
the file saying the same.

**Verify**: `GOWORK=off go build ./cmd/setup` → exit 0.

### Step 3: Static verification

Run build, vet, format check, and lint from the commands table. Commit.

**Verify**: each command gives its expected result.

### Step 4: Behavior checks in a disposable worktree

```bash
CHECK="$(mktemp -d)/setup-check"
git worktree add --detach "$CHECK" HEAD
```

1. **Scaffolded module.** Simulate what `gonew` does, in `$CHECK`:
   - Change line 1 of `go.mod` to `module example.com/acme/widget`.
   - Replace `github.com/aarock1234/go-template/` with
     `example.com/acme/widget/` in the import paths of every `.go` file under
     `cmd/` and `pkg/` (imports only; leave the constant you added in
     `cmd/setup/module.go` alone).
   - Run `go run ./cmd/setup -y` in the background.
   → exits 0 and prints `setup complete`;
   `grep -n "local-prefixes" -A1 .golangci.yml` shows
   `- example.com/acme/widget`;
   `grep -rn "aarock1234/go-template" .golangci.yml` → no matches;
   `go build ./...` exits 0.
2. **Template itself.** Reset the worktree
   (`git -C "$CHECK" reset --hard && git -C "$CHECK" clean -fd`), then run
   `go run ./cmd/setup -y` there.
   → exits 0; `git -C "$CHECK" diff --stat .golangci.yml` → no output.

Remove the worktree:

```bash
git worktree remove --force "$CHECK"
```

**Verify**: both checks behave as stated, and `git worktree list` no longer
shows the throwaway checkout.

## Test plan

No tests. This repository's rules forbid adding, running, or proposing tests
unless the maintainer explicitly asks. Verification is the static commands
and the two behavior checks in a disposable worktree.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `GOWORK=off go build ./...` and `GOWORK=off go vet ./...` exit 0
- [ ] `gofmt -l cmd` prints nothing
- [ ] Lint reports no issue that was not in the baseline
- [ ] `cmd/setup/module.go` exists
- [ ] `git diff .golangci.yml` is empty in your working checkout
- [ ] Both behavior checks in Step 4 produced the stated results
- [ ] `git worktree list` shows no leftover throwaway worktree
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- `apply` in `cmd/setup/main.go` still calls `warn` (plan 005 has not
  landed).
- `.golangci.yml` no longer contains the template's module path.
- You are about to run the setup tool anywhere other than the disposable
  worktree.
- The fix appears to require editing `.golangci.yml` in the template itself.
- A verification fails twice after a reasonable fix attempt.

## Maintenance notes

- The behavior relies on `gonew` leaving Go string literals unchanged. If a
  future scaffolding tool rewrites them, the constant would equal the new
  module path and the rewrite would silently do nothing; the scaffolded-module
  check in Step 4 is how to detect that.
- Another file that hardcodes the module path outside Go sources is one more
  entry in the list in `module.go`.
- If the template repository is ever renamed, update the constant.

Standards: Meet the solution standards in ~/.agents/AGENTS.md: idiomatic, beautiful, elegant, seamless, well-structured, dynamic, and future-proof, as explained in ~/.agents/CODING.md.
Guidance: Before implementing, read the repository's AGENTS.md and STYLE.md (if present in your checkout), load the go-style skill (`.agents/skills/go-style/SKILL.md`), and read all files in `cmd/setup/` in full.
Review: Read every changed file from top to bottom for readability before declaring the plan done.
Evidence: Verify with build, vet, formatting checks, lint, command output, and the relevant file contents. Do not add or run tests unless tests are explicitly requested.
Execution: Run every shell command that may take more than a few seconds, and every subagent task, in the background, and keep working on what does not depend on it until its completion notification arrives. Never sleep or poll.
