# Plan 015: The template pins the current fphttp release

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 3112ea3..HEAD -- go.mod go.sum pkg/client pkg/template`
> Plans 001 and 014 are expected to have changed files under `pkg/client`
> and possibly moved `golang.org/x/net` to the direct block of `go.mod`.
> `go.mod` must still pin fphttp at `v2.0.0`; otherwise treat it as a STOP
> condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW to MED (a minor-version bump of the library the client is built on)
- **Depends on**: plans/001-cookie-jar-domain-validation.md, plans/014-client-and-middleware-edge-cases.md (same package and `go.mod`; land them first)
- **Category**: migration
- **Planned at**: commit `3112ea3`, 2026-11-23

## Why this matters

The HTTP client in `pkg/client` is built on
`github.com/aarock1234/fphttp/v2`, which supplies the browser fingerprints.
`go.mod` pins `v2.0.0`, while the library's own repository is at `v2.1.3`.
The README promises that the client "sends requests as the newest Chrome
release"; the set of known releases ships inside the library, so a stale pin
means every project scaffolded today presents an older browser than the
documentation says. The maintainer does not notice locally because a
git-ignored `go.work` on their machine substitutes a sibling checkout of the
library. Bumping the pin makes CI, scaffolded projects, and the maintainer's
machine agree.

## Current state

- `go.mod:8`:

  ```
  	github.com/aarock1234/fphttp/v2 v2.0.0
  ```

- The packages that import fphttp: `pkg/client` (all six files, as
  `http "github.com/aarock1234/fphttp/v2"`) and `pkg/template`.

- `go.work` at the repository root is git-ignored and local to the
  maintainer's machine. It contains `use ( . ..\fphttp )`. It may not exist
  in your checkout. With it present, Go ignores the `go.mod` pin for fphttp,
  which hides exactly the thing this plan changes — so every Go command in
  this plan must run with `GOWORK=off`.

- CI runs `go mod tidy -diff` (`.github/workflows/ci.yaml:23-24`), so
  `go.mod` and `go.sum` must be tidy.

## Commands you will need

| Purpose          | Command                                                                                | Expected on success            |
|------------------|----------------------------------------------------------------------------------------|--------------------------------|
| Available update | `GOWORK=off go list -m -u github.com/aarock1234/fphttp/v2`                             | prints the pinned version and, in brackets, the newest one |
| Upgrade          | `GOWORK=off go get github.com/aarock1234/fphttp/v2@<version>`                          | exit 0                         |
| Tidy             | `GOWORK=off go mod tidy`                                                               | exit 0                         |
| Tidy check       | `GOWORK=off go mod tidy -diff`                                                         | exit 0, no output              |
| Build            | `GOWORK=off go build ./...`                                                            | exit 0, no output              |
| Vet              | `GOWORK=off go vet ./...`                                                              | exit 0, no output              |
| Lint             | `GOWORK=off go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run` | exit 0, `0 issues.`            |

These need network access. Run `go list -m -u`, `go get`, and lint in the
background.

## Scope

**In scope** (the only files you should modify):

- `go.mod`
- `go.sum`

**Out of scope** (do NOT touch, even though they look related):

- `go.work`, `go.work.sum` — local files that belong to the maintainer.
- Any `.go` file. If the new version needs source changes, that is a STOP
  condition, not something to patch here.
- Other direct dependencies (`chi`, `pgx`, `huh`, `lipgloss`, and so on). Do
  not run `go get -u` or upgrade anything else by hand. Indirect
  requirements that `go get` raises because the new fphttp needs them are
  expected and fine.
- `README.md`.
- Uncommitted maintainer work in `README.md`, `AGENTS.md`, `STYLE.md`,
  `pkg/service/service.go`, and `.agents/skills/go-style/`. Do not stash,
  revert, commit, or reformat those files.

## Git workflow

- Branch: `advisor/015-bump-fphttp`
- One commit. Message style is a short imperative sentence, as in `git log`:
  "Use fphttp v2 profiles in the client". Suggested:
  `Bump fphttp to v2.1.3` (use the version you actually pin).
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Find the newest published v2 release

Run the "Available update" command.

**Verify**: the output has the form
`github.com/aarock1234/fphttp/v2 v2.0.0 [v2.x.y]`. Record `v2.x.y`. It is
expected to be `v2.1.3` or newer. If there are no brackets, no newer version
is published: STOP.

### Step 2: Upgrade and tidy

Run the "Upgrade" command with the recorded version, then the "Tidy"
command.

**Verify**: `grep -n "aarock1234/fphttp/v2" go.mod` → one line showing the
recorded version; `GOWORK=off go mod tidy -diff` → exit 0, no output.

### Step 3: Review what else moved

Run `git diff go.mod`.

**Verify**: the only changed direct requirement is fphttp. Any other changed
line is in the `// indirect` block (or is `golang.org/x/net`, if plan 001
made it direct and the new fphttp requires a newer one). List every changed
module and its old and new version in your report.

### Step 4: Build, vet, and lint against the new version

Run build, vet, and lint from the commands table.

**Verify**: each command gives its expected result. Lint must report no
issue that the baseline (the same command at the commit before your change)
did not.

## Test plan

No tests. This repository's rules forbid adding, running, or proposing tests
unless the maintainer explicitly asks. Verification is the build, vet, tidy,
and lint commands with the workspace file disabled.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -c "aarock1234/fphttp/v2 v2.0.0" go.mod` prints `0`
- [ ] `GOWORK=off go mod tidy -diff` exits 0 with no output
- [ ] `GOWORK=off go build ./...` and `GOWORK=off go vet ./...` exit 0
- [ ] Lint reports no issue that was not in the baseline
- [ ] `git diff --stat` lists only `go.mod` and `go.sum`
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The module proxy cannot be reached, or no version newer than `v2.0.0` is
  published.
- The build or vet fails after the upgrade. Report the compiler errors in
  full; they describe the API changes the maintainer needs to decide on. Do
  not edit `.go` files.
- A direct requirement other than fphttp changes version.
- `go mod tidy` wants to remove or add a direct requirement unrelated to
  this bump.

## Maintenance notes

- Worth adding to the maintainer's release routine: bump fphttp in the
  template whenever fphttp ships new browser profiles, since the README's
  "newest Chrome" promise depends on it.
- fphttp selects an untagged commit of `github.com/refraction-networking/utls`
  (a pseudo-version in the indirect block). That choice belongs to fphttp;
  nothing to do here beyond noticing if it changes.
- After this lands, the maintainer's local `go.work` and CI build against the
  same fphttp only as long as the sibling checkout stays on the pinned tag.

Standards: Meet the solution standards in ~/.agents/AGENTS.md: idiomatic, beautiful, elegant, seamless, well-structured, dynamic, and future-proof, as explained in ~/.agents/CODING.md.
Guidance: Before implementing, read the repository's AGENTS.md and STYLE.md (if present in your checkout), and read `go.mod` in full.
Review: Read every changed file from top to bottom for readability before declaring the plan done.
Evidence: Verify with build, vet, tidy checks, lint, command output, and the relevant file contents. Do not add or run tests unless tests are explicitly requested.
Execution: Run every shell command that may take more than a few seconds, and every subagent task, in the background, and keep working on what does not depend on it until its completion notification arrives. Never sleep or poll.
