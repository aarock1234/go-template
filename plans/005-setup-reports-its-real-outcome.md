# Plan 005: Setup fails loudly instead of reporting a success it did not achieve

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 3112ea3..HEAD -- cmd/setup`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: LOW
- **Depends on**: none (plans 006, 007, 008, and 012 build on this one)
- **Category**: bug
- **Planned at**: commit `3112ea3`, 2026-11-23

## Why this matters

`cmd/setup` is a one-shot tool: a user scaffolds a project, runs
`make setup`, and the tool removes unwanted features, strips its marker
comments, tidies the module, and deletes itself. Because it runs once and
cannot be rerun, a false "setup complete" is expensive. Today it prints that
message and exits 0 in four situations where it did not do what was asked:

1. Run from any directory other than the project root, it finds no files,
   changes nothing, and reports success.
2. Interrupted with Ctrl-C, or with `go mod tidy` failing, it prints a
   warning and still reports success, leaving `go.mod` untidy.
3. A misspelled marker tag in a project file is silently stripped and its
   content always kept.
4. Contradictory flags such as `-postgres=false -postgres-hosting=external`
   are silently ignored.

After this plan each of those ends with a clear error and a non-zero exit.

## Current state

Files, all in `cmd/setup/`:

- `main.go` — flag parsing (`parseFlags`), orchestration (`apply`), `runGo`,
  `warn`.
- `feature.go` — the `tag` type and constants, removal specs, `config`.
- `marker.go` — marker parsing (`parseMarkers`, `parseLine`) and `render`.
- `fs.go` — `loadMarkedFiles`, `rewriteFiles`, `removePaths`.
- `form.go` — the interactive form. Not changed by this plan.

How markers work: project files mark feature-specific content with comments.
A block runs from a full-line `# [tag]` or `// [tag]` to the matching
`[/tag]`. A trailing comment made only of tags, as in
`"net/http" // [server]`, marks one line.

Missing files are skipped, `cmd/setup/fs.go:18-28`:

```go
// loadMarkedFiles reads and parses each file in paths, skipping files that
// do not exist. It reads everything before setup changes anything, so a
// malformed marker aborts setup with the project untouched.
func loadMarkedFiles(paths []string) ([]markedFile, error) {
	var files []markedFile

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
```

The marked files, `cmd/setup/feature.go:62-65`. All four exist in a freshly
scaffolded project, and setup runs only once:

```go
var markedFiles = []string{fileMain, fileMakefile, fileEnvExample, fileCompose}
```

The tags, `cmd/setup/feature.go:13-22`:

```go
// Marker tags used across the project. Define new tags here when adding
// marked sections to project files.
const (
	tagServer         tag = "server"
	tagClient         tag = "client"
	tagDocker         tag = "docker"
	tagPostgres       tag = "postgres"
	tagPostgresDocker tag = "postgres-docker"
	tagSetup          tag = "setup"
)
```

Any tag name parses, `cmd/setup/marker.go:56-77` (excerpt):

```go
func parseMarkers(data string) ([]markedLine, error) {
	var (
		lines []markedLine
		open  []int // indexes of the open block markers, innermost last
	)

	for i, raw := range strings.Split(data, "\n") {
		l := parseLine(raw)

		switch l.marker {
		case markerOpen:
			open = append(open, i)
		case markerClose:
			if len(open) == 0 || lines[open[len(open)-1]].tags[0] != l.tags[0] {
				return nil, fmt.Errorf("line %d: [/%s] does not close the innermost open block", i+1, l.tags[0])
			}

			open = open[:len(open)-1]
		}

		lines = append(lines, l)
	}
```

The outcome is reported regardless of tidy or interruption,
`cmd/setup/main.go:148-178`:

```go
// apply removes everything c disables, strips the remaining markers, deletes
// the setup command, and tidies the module. Malformed markers abort before
// anything changes. A failed file or directory change does not stop the
// remaining steps but makes apply return an error. Failures of go mod tidy
// and go fmt are only warnings.
func apply(ctx context.Context, c config) error {
	removal := c.removal()

	files, err := loadMarkedFiles(markedFiles)
	if err != nil {
		return err
	}

	// Rewrite before removing paths so that a marked file being deleted,
	// such as compose.yaml, is not recreated.
	changeErr := errors.Join(
		rewriteFiles(files, removal.tags),
		removePaths(removal.paths),
	)

	warn(runGo(ctx, "mod", "tidy"))
	warn(runGo(ctx, "fmt", "./..."))

	if changeErr != nil {
		return fmt.Errorf("setup incomplete: %w", changeErr)
	}

	fmt.Println("\nsetup complete")

	return nil
}
```

`runGo` uses `exec.CommandContext(ctx, ...)`, so after Ctrl-C both commands
fail. `warn` (`main.go:194-199`) prints to stderr and is used nowhere else.

The hosting flag is accepted unconditionally, `cmd/setup/main.go:99-100`:

```go
	flags.BoolVar(&c.postgres, "postgres", true, "keep PostgreSQL support")
	flags.TextVar(&c.pgHosting, "postgres-hosting", hostingDocker, "PostgreSQL `hosting`: docker or external")
```

and then silently overridden, `cmd/setup/feature.go:213-218`:

```go
	switch {
	case !c.postgres:
		features = append(features, postgresRemoval)
	case !c.docker || c.pgHosting == hostingExternal:
		features = append(features, postgresDockerRemoval)
	}
```

Conventions that apply (from the repository's `AGENTS.md` and `STYLE.md`):

- Every nonempty struct literal is multiline with keyed fields, even with one
  field.
- Doc comments state each function's contract; update them when the contract
  changes. `cmd/setup/main.go:1-29` is the package comment and documents the
  flags and marker grammar.
- Error strings are lowercase with no trailing punctuation, wrapped with
  `fmt.Errorf("context: %w", err)`.
- Messages printed for the user are lowercase.

## Commands you will need

A local, git-ignored `go.work` may exist on the maintainer's machine. Prefix
Go commands with `GOWORK=off` so results match CI.

| Purpose      | Command                                                                                | Expected on success |
|--------------|----------------------------------------------------------------------------------------|---------------------|
| Build        | `GOWORK=off go build ./...`                                                            | exit 0, no output   |
| Vet          | `GOWORK=off go vet ./...`                                                              | exit 0, no output   |
| Format check | `gofmt -l cmd`                                                                         | no output           |
| Lint         | `GOWORK=off go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run` | exit 0, `0 issues.` |

Run lint once before changing anything to learn the baseline; if it already
reports issues, record them and require only that you add none. Run lint in
the background.

**Never run `go run ./cmd/setup` or `make setup` in your working checkout.**
The tool deletes files, including itself. Behavior checks run only in a
disposable worktree, as described in Step 6.

## Scope

**In scope** (the only files you should modify):

- `cmd/setup/main.go`
- `cmd/setup/feature.go`
- `cmd/setup/marker.go`
- `cmd/setup/fs.go`

**Out of scope** (do NOT touch, even though they look related):

- `cmd/setup/form.go` — the form already hides the hosting question when it
  does not apply.
- The marker grammar itself (the three regular expressions in `marker.go`).
- Making file rewrites atomic or keeping `cmd/setup` when a write fails.
  Considered and deferred: a failed write to four small files in the working
  directory is very unlikely, and the tool cannot be rerun usefully after
  markers are stripped anyway.
- `Makefile`, `README.md`, and every marked project file.
- Any test file. This repository forbids adding or running tests unless the
  maintainer asks for them.
- Uncommitted maintainer work in `README.md`, `AGENTS.md`, `STYLE.md`,
  `pkg/service/service.go`, and `.agents/skills/go-style/`. Do not stash,
  revert, commit, or reformat those files.

## Git workflow

- Branch: `advisor/005-setup-reports-its-real-outcome`
- One commit per step is fine, or one commit overall. Message style is a
  short imperative sentence, as in `git log`: "Use fphttp v2 profiles in the
  client", "Address review findings". Example:
  `Make setup fail when run outside the project root`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: A missing marked file is an error

In `cmd/setup/fs.go`, remove the `fs.ErrNotExist` branch from
`loadMarkedFiles` so a missing file returns an error. Make the error tell the
user what to do, for example:
`read cmd/template/main.go: <os error>: run setup from the project root`.
Remove the `errors` and `io/fs` imports if they become unused. Update the
function's doc comment: it no longer skips missing files, and a missing file
aborts setup with the project untouched.

This single change is the working-directory check: outside the project root
the first marked file is missing.

**Verify**: `GOWORK=off go build ./cmd/setup` → exit 0, and
`grep -n "ErrNotExist" cmd/setup/fs.go` → no matches.

### Step 2: Unknown marker tags are an error

In `cmd/setup/feature.go`, directly below the tag constants, declare the list
of known tags and update the constants' comment to say a new tag must be
added to both:

```go
// knownTags lists every tag a project file may use. A marker with any other
// tag is a mistake and aborts setup.
var knownTags = []tag{
	tagServer,
	tagClient,
	tagDocker,
	tagPostgres,
	tagPostgresDocker,
	tagSetup,
}
```

In `cmd/setup/marker.go`, make `parseMarkers` return an error for the first
line whose marker carries a tag that is not in `knownTags`, in the same style
as its existing errors: `line %d: unknown tag [%s]`. Apply this to block
markers and inline markers alike. Update the `parseMarkers` doc comment to
mention the new error. Keep the check small; a helper on `markedLine` that
returns the first unknown tag is one readable way to do it.

**Verify**: `GOWORK=off go build ./cmd/setup` → exit 0, and
`grep -n "unknown tag" cmd/setup/marker.go` → at least one match.

### Step 3: A failed or interrupted tidy is an error

In `cmd/setup/main.go`, change `apply` so the results of `go mod tidy` and
`go fmt ./...` are joined into one error instead of being printed as
warnings. Return in this order:

1. If the file and directory changes failed: `setup incomplete: <error>`
   (as today).
2. Else if tidying failed: an error that says the features were removed but
   the module was not tidied, and names the two commands to run by hand:
   `go mod tidy` and `go fmt ./...`.
3. Else print `setup complete` and return nil.

Delete the now-unused `warn` function. Update the `apply` doc comment: its
last sentence currently says tidy and fmt failures are only warnings.

**Verify**: `GOWORK=off go build ./cmd/setup` → exit 0, and
`grep -n "func warn" cmd/setup/main.go` → no matches.

### Step 4: Contradictory hosting flags are an error

In `parseFlags` in `cmd/setup/main.go`, after `flags.Parse` succeeds, find
out whether `-postgres-hosting` was given explicitly. `flag.FlagSet.Visit`
calls its function only for flags that were set; use it. When the flag was
set explicitly:

- with `-postgres=false`, return an error such as
  `-postgres-hosting requires -postgres`;
- with `-docker=false` and the value `docker`, return an error such as
  `-postgres-hosting=docker requires -docker`.

Keep `parseFlags` readable: extract the two checks into a small function
with a doc comment if the body grows. Do not change `config.disabled`.

Add one sentence to the package comment at the top of `main.go` (after "At
least one of the server or the client must be kept.") stating that
`-postgres-hosting` is rejected when PostgreSQL is not kept, and that
`docker` hosting is rejected when Docker is not kept.

**Verify**: `GOWORK=off go build ./cmd/setup` → exit 0.

### Step 5: Static verification

Run build, vet, format check, and lint from the commands table. Commit your
work so the next step can check it out elsewhere.

**Verify**: each command gives its expected result; `git status --short cmd`
is empty after the commit.

### Step 6: Behavior checks in a disposable worktree

Create a throwaway checkout of your commit, outside the repository:

```bash
CHECK="$(mktemp -d)/setup-check"
git worktree add --detach "$CHECK" HEAD
```

Run each check below from the stated directory. `go run` prints
`exit status 1` and itself exits 1 when the program exits non-zero. The first
`go run` may download modules; run it in the background.

1. **Wrong directory.** In `$CHECK/pkg`: `go run ../cmd/setup -y`
   → exits non-zero; stderr contains `run setup from the project root`.
   Then `git -C "$CHECK" status --short` → no output.
2. **Contradictory flags.** In `$CHECK`:
   `go run ./cmd/setup -y -postgres=false -postgres-hosting=external`
   → exits non-zero; stderr mentions `-postgres-hosting`.
   Then `git -C "$CHECK" status --short` → no output.
3. **Unknown tag.** In `$CHECK`, append two lines to `Makefile`:
   `# [bogus]` and `# [/bogus]`. Then `go run ./cmd/setup -y`
   → exits non-zero; stderr contains `unknown tag [bogus]`.
   `git -C "$CHECK" status --short` → only ` M Makefile`.
   Restore it: `git -C "$CHECK" checkout Makefile`.
4. **Happy path.** In `$CHECK`: `go run ./cmd/setup -y`
   → exits 0 and prints `setup complete`. Then `test ! -d cmd/setup` exits 0
   and `go build ./...` exits 0.

Remove the worktree when finished:

```bash
git worktree remove --force "$CHECK"
```

**Verify**: all four checks behave as stated, and `git worktree list` no
longer shows the throwaway checkout.

## Test plan

No tests. This repository's rules forbid adding, running, or proposing tests
unless the maintainer explicitly asks. Verification is the static commands
and the four behavior checks in a disposable worktree.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `GOWORK=off go build ./...` and `GOWORK=off go vet ./...` exit 0
- [ ] `gofmt -l cmd` prints nothing
- [ ] Lint reports no issue that was not in the baseline
- [ ] `grep -n "ErrNotExist" cmd/setup/fs.go` returns no matches
- [ ] `grep -n "func warn" cmd/setup/main.go` returns no matches
- [ ] `grep -n "knownTags" cmd/setup/feature.go cmd/setup/marker.go` matches in both files
- [ ] The four behavior checks in Step 6 produced the stated results
- [ ] `git worktree list` shows no leftover throwaway worktree
- [ ] `git diff --stat 3112ea3..HEAD` lists only files under `cmd/setup/` and `plans/`
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The code in `cmd/setup` does not match the excerpts above.
- You are about to run the setup tool anywhere other than the disposable
  worktree.
- The unknown-tag check rejects a marker that exists in the repository's
  own files at the planned commit (that would mean a tag is in use that this
  plan does not know about). Report the file, line, and tag.
- The happy-path check fails because `go mod tidy` cannot reach the network.
  Report it; do not weaken the new error.
- A verification fails twice after a reasonable fix attempt.

## Maintenance notes

- A new feature tag now needs two edits in `feature.go`: the constant and
  `knownTags`. Forgetting the second is caught on the first setup run with
  an "unknown tag" error.
- A trailing comment made only of bracket tokens in a marked file, such as
  `x := 1 // [note]`, is now an error instead of being silently stripped.
- Reviewers should confirm that the error for an untidied module names the
  recovery commands, since the tool has already deleted itself by then.
- Plans 006, 007, 008, and 012 edit the same files and assume this plan has
  landed.

Standards: Meet the solution standards in ~/.agents/AGENTS.md: idiomatic, beautiful, elegant, seamless, well-structured, dynamic, and future-proof, as explained in ~/.agents/CODING.md.
Guidance: Before implementing, read the repository's AGENTS.md and STYLE.md (if present in your checkout), load the go-style skill (`.agents/skills/go-style/SKILL.md`), and read all five files in `cmd/setup/` in full.
Review: Read every changed file from top to bottom for readability before declaring the plan done.
Evidence: Verify with build, vet, formatting checks, lint, command output, and the relevant file contents. Do not add or run tests unless tests are explicitly requested.
Execution: Run every shell command that may take more than a few seconds, and every subagent task, in the background, and keep working on what does not depend on it until its completion notification arrives. Never sleep or poll.
