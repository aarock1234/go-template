# Plan 007: After setup, the README describes only the features that were kept

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 3112ea3..HEAD -- cmd/setup README.md`
> Plans 005 and 006 are expected to have changed `cmd/setup`. Confirm their
> changes are present (see "Current state"). `README.md` must match the
> excerpts below; on a mismatch, treat it as a STOP condition.
>
> **Precondition (run second)**: `git status --short README.md` must print
> nothing. If it prints ` M README.md`, the maintainer has uncommitted edits
> to the file this plan rewrites: STOP and ask the operator to commit them
> first.

## Status

- **Priority**: P2
- **Effort**: M to L
- **Risk**: LOW (documentation and a one-shot tool; nothing at runtime)
- **Depends on**: plans/005-setup-reports-its-real-outcome.md (required), plans/006-setup-rewrites-module-identity.md (same files; land it first)
- **Category**: docs
- **Planned at**: commit `3112ea3`, 2026-11-23

## Why this matters

A user scaffolds a project from this template and runs the one-shot setup
tool, which deletes the features they do not want. The tool cleans
`cmd/template/main.go`, `Makefile`, `.env.example`, and `compose.yaml`, but
never the README. Every scaffold except "keep everything" therefore ships a
README that documents Makefile targets, HTTP routes, environment variables,
and packages that no longer exist — `make db` fails with "no rule to make
target" — and still tells the reader to run `make setup`, which has deleted
itself. After this plan the README is one more marked file and comes out of
setup accurate.

## Current state

### The setup tool (`cmd/setup/`)

Project files mark feature-specific content with comments. A block runs from
a full-line `# [tag]` or `// [tag]` to the matching `[/tag]`; blocks may
nest. A trailing comment made only of tags, as in `"net/http" // [server]`,
marks one line, which is removed when **any** of its tags is removed. After
removing disabled features, setup strips every remaining marker.

The grammar, `cmd/setup/marker.go:20-33`:

```go
var (
	// blockMarker matches a trimmed line that holds only a block marker
	// comment, such as "# [postgres]" or "// [/server]".
	blockMarker = regexp.MustCompile(`^(?:#|//)\s*\[(/?)([a-z0-9-]+)\]$`)

	// inlineMarker matches code followed by a trailing comment made only of
	// tags, such as `"net/http" // [server]` or "PORT=8080 # [a] [b]". The
	// first group is the code and the second holds the tags.
	inlineMarker = regexp.MustCompile(`^(.*?\S)\s+(?:#|//)((?:\s*\[[a-z0-9-]+\])+)\s*$`)

	// tagToken matches a single "[tag]" in an inline marker, capturing the
	// tag name.
	tagToken = regexp.MustCompile(`\[([a-z0-9-]+)\]`)
)
```

The marked files, `cmd/setup/feature.go:62-65`:

```go
var markedFiles = []string{fileMain, fileMakefile, fileEnvExample, fileCompose}
```

The infrastructure tags, `cmd/setup/feature.go:15-22`: `server`, `client`,
`docker`, `postgres`, `postgres-docker`, `setup`. After plan 005 the same
file declares `knownTags`, a slice of those six, and `parseMarkers` rejects
any marker whose tag is not in it. If `knownTags` does not exist, plan 005
has not landed: STOP.

Which tag is removed when:

| Tag               | Removed when                                                        |
|-------------------|---------------------------------------------------------------------|
| `server`          | `-server=false`                                                     |
| `client`          | `-client=false`                                                     |
| `docker`          | `-docker=false`                                                     |
| `postgres`        | `-postgres=false`                                                   |
| `postgres-docker` | `-postgres=false`, or `-docker=false`, or `-postgres-hosting=external` |
| `setup`           | always (setup deletes itself)                                       |

Optional packages have no tags today. They are removed by path,
`cmd/setup/feature.go:108-149` (first two of six entries shown; the others
are `state`, `cycle`, `fake`, `ptr`, each with path `pkg/<name>`):

```go
// optionalPackage describes a selectable package shown in the setup form.
type optionalPackage struct {
	name    string  // identifies the package in the -packages flag
	label   string  // describes the package in the form
	feature feature // removes the package when it is deselected
}

// optionalPackages is the single source of truth for all toggleable
// packages. Add new entries here; the form, the -packages flag, and removal
// derive from it automatically.
var optionalPackages = []optionalPackage{
	{
		name:    "worker",
		label:   "Worker Pool: bounded concurrency via errgroup",
		feature: feature{paths: []string{"pkg/worker"}},
	},
	{
		name:    "retry",
		label:   "Retry: exponential backoff with jitter",
		feature: feature{paths: []string{"pkg/retry"}},
	},
```

They are consumed in `config.disabled`, `cmd/setup/feature.go:220-224`:

```go
	for _, p := range optionalPackages {
		if !slices.Contains(c.packages, p.name) {
			features = append(features, p.feature)
		}
	}
```

`feature` is `struct { paths []string; tags []tag }`
(`cmd/setup/feature.go:67-71`).

### The README (`README.md`)

Sections, in order: title and badges; intro paragraph; `## Quick Start`
(install `gonew`, scaffold, run `make setup`, flag list); `## Setup`
(prerequisites and a four-item numbered list); `## Running`;
`## Project Structure` (a tree inside a fenced code block);
`## Development` (a command table, then `### Database`, `### Docker`,
`### HTTP Endpoints`, `### Releases`); `## Configuration` (a variable table,
then three paragraphs about the HTTP client); `## Style Guide`;
`## License`.

The project tree is a fenced code block, `README.md:66-92`:

```
go-template/
├── .agents/skills/     go-style coding guide for AI agents
├── cmd/
│   ├── setup/          interactive feature selection (deletes itself)
│   └── template/       entrypoint: config, dependencies, workloads
├── pkg/
│   ├── apperr/         application error types and sentinels
│   ├── client/         fphttp client: TLS/HTTP2 fingerprints, cookies, redirects, proxies
...
├── Dockerfile          multi-stage: dev, builder (multi-arch), production (Alpine, non-root)
├── compose.yaml        app and PostgreSQL with docker compose watch
└── Makefile            run `make` to list targets
```

An example table, `README.md:108-116`:

```markdown
| Command                        | Description                            |
| ------------------------------ | -------------------------------------- |
| `make db`                      | Start PostgreSQL only (localhost:5432) |
| `make db-down`                 | Stop PostgreSQL                        |
| `make migrate`                 | Apply migrations                       |
| `make migrate-down`            | Roll back the last migration           |
| `make migrate-new name=<name>` | Create a migration file                |

To use an external database, choose external hosting during setup and set `DATABASE_URL` in `.env`.
```

### Two Markdown facts this plan relies on

- An HTML comment (`<!-- ... -->`) is invisible in rendered Markdown **except
  inside a fenced code block**, where it is shown literally. Markers must
  therefore never sit inside a code fence.
- A line between two table rows ends the table. Rows are therefore marked
  with a trailing inline marker after the row's final pipe, never with block
  markers between rows. GitHub ignores the extra trailing cell.

### Conventions that apply (from `AGENTS.md` and `STYLE.md`)

- Every nonempty struct literal is multiline with keyed fields, even with one
  field.
- Derive behavior from one source of truth; adding a case should mean adding
  data in one place.
- Doc comments state each contract; update them when the contract changes.
  The package comment at `cmd/setup/main.go:1-29` documents the marker
  grammar.

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
Behavior checks run only in a disposable worktree (Step 6).

## Scope

**In scope** (the only files you should modify):

- `cmd/setup/marker.go` (grammar and its comments)
- `cmd/setup/feature.go` (`markedFiles`, optional-package tags, `knownTags`)
- `cmd/setup/main.go` (package comment only)
- `README.md`

**Out of scope** (do NOT touch, even though they look related):

- `STYLE.md` and `AGENTS.md`. Both mention setup markers and cite removable
  files, but they are untracked work in progress by the maintainer. Do not
  create, edit, or commit them.
- The README's title, license badge, CI badge, intro paragraph, the
  `### Releases` section, `## Style Guide`, and `## License`.
- `cmd/setup/form.go`, `cmd/setup/fs.go`, `cmd/setup/module.go`.
- Rewording README content beyond what the steps below require.
- Any test file. This repository forbids adding or running tests unless the
  maintainer asks for them.
- Uncommitted maintainer work in `pkg/service/service.go` and
  `.agents/skills/go-style/`. Do not stash, revert, commit, or reformat it.

## Git workflow

- Branch: `advisor/007-setup-prunes-the-readme`
- Commit per step or per logical unit. Message style is a short imperative
  sentence, as in `git log`: "Use fphttp v2 profiles in the client". Example:
  `Accept HTML comment markers in setup`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Accept HTML comments as markers

In `cmd/setup/marker.go`, extend the two expressions so `<!--` is a third
comment opener and an optional ` -->` closes it:

```go
blockMarker  = regexp.MustCompile(`^(?:#|//|<!--)\s*\[(/?)([a-z0-9-]+)\](?:\s*-->)?$`)
inlineMarker = regexp.MustCompile(`^(.*?\S)\s+(?:#|//|<!--)((?:\s*\[[a-z0-9-]+\])+)(?:\s*-->)?\s*$`)
```

The capture groups keep their numbers, so `parseLine` needs no change.
Update the comments on both variables with an HTML example each, such as
`<!-- [docker] -->` and ``| `make db` | ... | <!-- [postgres-docker] -->``.
Update the marker-grammar paragraph of the package comment in
`cmd/setup/main.go` to mention the `<!-- [tag] -->` form and that it exists
for Markdown.

**Verify**: `GOWORK=off go build ./cmd/setup` → exit 0.

### Step 2: Give every optional package a tag derived from its name

In `cmd/setup/feature.go`:

- Reduce `optionalPackage` to `name` and `label`. Document that `name`
  identifies the package in the `-packages` flag, names its directory under
  `pkg/`, and is its marker tag.
- Add a method that returns the package's removal spec, derived from the
  name:

  ```go
  // removal returns what to delete when the package is deselected: its
  // directory and every section marked with its name.
  func (p optionalPackage) removal() feature {
  	return feature{
  		paths: []string{"pkg/" + p.name},
  		tags:  []tag{tag(p.name)},
  	}
  }
  ```

- Remove the `feature:` field from all six entries of `optionalPackages`.
- In `config.disabled`, append `p.removal()` instead of `p.feature`.
- Make the known-tag set include one tag per optional package, derived from
  `optionalPackages` so that adding a package needs no second edit. Replacing
  the `knownTags` variable with a function of the same name that returns the
  six infrastructure tags followed by the package tags is one readable way;
  update its caller in `marker.go` and both doc comments accordingly.

**Verify**: `GOWORK=off go build ./cmd/setup` → exit 0, and
`grep -n "feature:" cmd/setup/feature.go` → no matches.

### Step 3: Make the README a marked file

In `cmd/setup/feature.go`, add `fileReadme = "README.md"` to the file-name
constant block and append `fileReadme` to `markedFiles`. If the slice literal
no longer fits comfortably on one line, put one element per line.

**Verify**: `GOWORK=off go build ./cmd/setup` → exit 0.

### Step 4: Restructure the two README parts that cannot be marked as written

1. **Project Structure.** Replace the fenced tree with a Markdown table with
   two columns, `Path` and `Purpose`, one row per current tree line, in the
   same order, with the same descriptions. Write paths in backticks with a
   trailing slash for directories (for example `` `pkg/retry/` ``,
   `` `cmd/setup/` ``, `` `Dockerfile` ``). Drop the `go-template/`, `cmd/`,
   and `pkg/` grouping lines; the full paths carry that information.
2. **Running.** The first code block mixes an optional database command with
   `make dev`. Split it so the database command can be removed on its own:
   a short sentence and a one-line `make db` block for the bundled
   PostgreSQL, then the `make dev` block. Keep the wording brief. Reword the
   Docker Compose sentence so it does not mention the bundled database
   ("With Docker Compose:").

Also change the sentence under the Database table to
``To use an external database, set `DATABASE_URL` in `.env`.`` — "choose
external hosting during setup" is meaningless once setup has run.

**Verify**: `grep -n "├──\|└──" README.md` → no matches.

### Step 5: Mark the README

Add markers exactly as listed. Block markers are full lines of the form
`<!-- [tag] -->` and `<!-- [/tag] -->`, placed outside code fences and
outside tables, with a blank line between a marker and adjacent prose,
headings, or fences. Inline markers go at the very end of a line, after one
space; on a table row that is after the final `|`.

Block markers (wrap the whole stated content, heading included):

| Content                                                                 | Tag               |
|-------------------------------------------------------------------------|-------------------|
| The entire `## Quick Start` section                                     | `setup`           |
| In `## Running`: the sentence and code block for `make db`              | `postgres-docker` |
| In `## Running`: the Docker Compose sentence and its code block         | `docker`          |
| The entire `### Database` section                                       | `postgres`        |
| The entire `### Docker` section                                         | `docker`          |
| The entire `### HTTP Endpoints` section                                 | `server`          |
| In `## Configuration`: the three paragraphs about the HTTP client, from "The HTTP client sends requests as…" through "Enable `client.WithInsecureSkipVerify(true)`…" | `client` |

Inline markers:

| Line                                                                    | Tags                    |
|-------------------------------------------------------------------------|-------------------------|
| `## Setup` list item "Replace `pkg/template` with your own domain logic" | `client`               |
| `## Setup` list item "Add migrations with …"                            | `postgres`              |
| Project Structure row `cmd/setup/`                                      | `setup`                 |
| Project Structure rows `pkg/apperr/`, `pkg/handler/`, `pkg/middleware/`, `pkg/respond/` | `server` |
| Project Structure rows `pkg/client/`, `pkg/template/`                   | `client`                |
| Project Structure row `pkg/db/`                                         | `postgres`              |
| Project Structure row `pkg/service/`                                    | `server` and `postgres` (written `<!-- [server] [postgres] -->`) |
| Project Structure rows `pkg/cycle/`, `pkg/fake/`, `pkg/ptr/`, `pkg/retry/`, `pkg/state/`, `pkg/worker/` | the package's own name, e.g. `retry` |
| Project Structure rows `Dockerfile`, `compose.yaml`                     | `docker`                |
| Database table rows `make db`, `make db-down`                           | `postgres-docker`       |
| HTTP Endpoints table rows `GET /readyz`, `GET /api/example`             | `postgres`              |
| Configuration table row `PROXY`                                         | `client`                |
| Configuration table row `DATABASE_URL`                                  | `postgres`              |
| Configuration table rows `PORT`, `READ_TIMEOUT`, `WRITE_TIMEOUT`, `SHUTDOWN_TIMEOUT` | `server`   |

These assignments mirror the removal specs in `cmd/setup/feature.go`
(`serverRemoval`, `clientRemoval`, `dockerRemoval`, `postgresRemoval`). If a
path in the Project Structure table is removed by a spec and is not listed
here, STOP and report it.

**Verify**:

- `grep -c "<!-- \[" README.md` → a positive number.
- Block markers are balanced:
  `grep -c "^<!-- \[[a-z-]*\] -->$" README.md` equals
  `grep -c "^<!-- \[/[a-z-]*\] -->$" README.md`.
- No marker is inside a code fence: read the file top to bottom once and
  confirm.

### Step 6: Static verification, then behavior checks in a disposable worktree

Run build, vet, format check, and lint from the commands table. Commit.
Then:

```bash
CHECK="$(mktemp -d)/setup-check"
git worktree add --detach "$CHECK" HEAD
```

1. **Minimal scaffold.** In `$CHECK`, run in the background:
   `go run ./cmd/setup -y -server=false -postgres=false -docker=false -packages=`
   → exits 0 and prints `setup complete`. Then, in `$CHECK`:
   - `grep -n 'make db\|make up\|make setup\|gonew\|/readyz\|DATABASE_URL\|READ_TIMEOUT\|retry/\|worker/\|handler/\|compose.yaml\|<!--' README.md`
     → no matches.
   - `grep -n 'PROXY' README.md` → at least one match (the client was kept).
   - `grep -n 'pkg/client/' README.md` → one match.
   - `go build ./...` → exit 0.
2. **Full scaffold.** Reset the worktree
   (`git -C "$CHECK" reset --hard && git -C "$CHECK" clean -fd`), then run
   `go run ./cmd/setup -y` there.
   → exits 0. Then:
   - `grep -n '<!--\|make setup\|gonew' README.md` → no matches.
   - `grep -n 'make db' README.md` → at least two matches.
   - `grep -n '/readyz' README.md` → one match.
   - `grep -n 'pkg/retry/' README.md` → one match.

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
- [ ] `grep -n "fileReadme" cmd/setup/feature.go` matches twice (constant and `markedFiles`)
- [ ] `grep -n "feature:" cmd/setup/feature.go` returns no matches
- [ ] `grep -n "├──\|└──" README.md` returns no matches
- [ ] Open and close block-marker counts in `README.md` are equal
- [ ] Both behavior checks in Step 6 produced the stated results
- [ ] `git worktree list` shows no leftover throwaway worktree
- [ ] `git status --short` shows no change to `STYLE.md`, `AGENTS.md`, or `.agents/` made by you
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- `git status --short README.md` shows uncommitted changes before you start.
- `knownTags` does not exist in `cmd/setup/feature.go` (plan 005 has not
  landed).
- The README's sections or tables differ from the description in "Current
  state".
- A marker would have to go inside a fenced code block or between two table
  rows to express what a step asks.
- After the full-scaffold check, the README differs from the committed
  README in any way other than: removed markers, the removed
  `## Quick Start` section, the removed `cmd/setup/` row, and collapsed blank
  lines.
- You are about to run the setup tool anywhere other than the disposable
  worktree.
- A verification fails twice after a reasonable fix attempt.

## Maintenance notes

- Reviewers should preview the committed `README.md` on GitHub (or any
  GitHub-flavored renderer) and confirm the markers are invisible and every
  table renders as a table. That visual check cannot be automated here.
- New README content about a removable feature must be marked, or it becomes
  stale documentation in scaffolds. New marked files are added to
  `markedFiles`.
- An optional package's name is now also its marker tag. A package named
  like an infrastructure tag (for example `server`) would collide.
- Deferred: `STYLE.md` cites `pkg/service/service.go`, `pkg/respond/respond.go`,
  and `pkg/state/` as examples, and both `STYLE.md` and `AGENTS.md` tell
  readers to preserve setup markers that no longer exist after setup. Once
  those files are committed they can be marked the same way.
- Deferred: the CI badge and title still name the template repository in a
  scaffolded project.

Standards: Meet the solution standards in ~/.agents/AGENTS.md: idiomatic, beautiful, elegant, seamless, well-structured, dynamic, and future-proof, as explained in ~/.agents/CODING.md.
Guidance: Before implementing, read the repository's AGENTS.md and STYLE.md (if present in your checkout), load the go-style skill (`.agents/skills/go-style/SKILL.md`), and read `README.md` and all files in `cmd/setup/` in full.
Review: Read every changed file from top to bottom for readability before declaring the plan done.
Evidence: Verify with build, vet, formatting checks, lint, command output, and the relevant file contents. Do not add or run tests unless tests are explicitly requested.
Execution: Run every shell command that may take more than a few seconds, and every subagent task, in the background, and keep working on what does not depend on it until its completion notification arrives. Never sleep or poll.
