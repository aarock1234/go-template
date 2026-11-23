# Plan 002: A local Go workspace file never reaches the Docker build

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 3112ea3..HEAD -- .dockerignore compose.yaml`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: dx
- **Planned at**: commit `3112ea3`, 2026-11-23

## Why this matters

`.gitignore` ignores `go.work` and `go.work.sum`, so developers are expected
to keep local workspace files. `.dockerignore` does not exclude them, and the
Dockerfile copies the whole directory into the image. A workspace file names
sibling directories that do not exist inside the container, so `go build`
fails there. `make up` and `make watch` therefore break on any machine that
has a `go.work`, including the maintainer's, whose workspace points at
`..\fphttp`. Excluding the two files makes container builds use `go.mod`
alone, the same as CI.

## Current state

- `.dockerignore` — lists what stays out of the build context. It has no
  workspace entry. Its sections look like this (`.dockerignore:1-13`):

  ```
  # Build artifacts
  *.exe
  *.dll
  *.so
  *.dylib
  *.test
  *.out
  bin/

  # Environment
  .env
  .env.*
  !.env.example
  ```

- `.gitignore:23-25`:

  ```
  # Go workspace file
  go.work
  go.work.sum
  ```

- `Dockerfile:8-11` — copies `go.mod` and `go.sum`, downloads modules, then
  `COPY . .`. Lines 17 and 27 run `go build`.

- `compose.yaml:20-30` — the watch configuration that syncs the source tree
  into the running container:

  ```yaml
      develop:
        watch:
          - action: sync+restart
            path: .
            target: /app
            ignore:
              - .git/
              - .agents/
              - bin/
          - action: rebuild
            path: go.mod
  ```

  The `ignore` list repeats entries that `.dockerignore` also has (`.git`,
  `.agents/`, `bin/`), so the repository's convention is to list them in both
  places.

`compose.yaml` contains setup markers: full-line comments such as
`# [server]` and `# [/server]`. Do not add, remove, or move any of them.

## Commands you will need

| Purpose          | Command                                         | Expected on success                    |
|------------------|-------------------------------------------------|----------------------------------------|
| Entry present    | `grep -n "go.work" .dockerignore`               | two matches: `go.work`, `go.work.sum`  |
| Compose is valid | `docker compose config --quiet`                 | exit 0, no output (needs Docker)       |
| Image builds     | `docker build --target builder -t go-template-check .` | exit 0 (needs Docker; slow)     |

The two Docker commands are optional evidence. Run them only if `docker` is
installed and its daemon is running; run the build in the background. If you
build the image, remove it afterwards with `docker image rm go-template-check`.

## Scope

**In scope** (the only files you should modify):

- `.dockerignore`
- `compose.yaml` (only the `ignore` list shown above)

**Out of scope** (do NOT touch, even though they look related):

- `go.work`, `go.work.sum` — local, git-ignored files that belong to the
  maintainer. Do not delete or edit them.
- `Makefile` — do not add `GOWORK=off`; the maintainer uses the workspace
  deliberately when developing `fphttp` alongside this template.
- `Dockerfile`, `.gitignore`.
- Any setup marker comment in `compose.yaml`.
- Uncommitted maintainer work in `README.md`, `AGENTS.md`, `STYLE.md`,
  `pkg/service/service.go`, and `.agents/skills/go-style/`. Do not stash,
  revert, commit, or reformat those files.

## Git workflow

- Branch: `advisor/002-keep-go-work-out-of-docker`
- One commit. Message style is a short imperative sentence, as in `git log`:
  "Use fphttp v2 profiles in the client". Suggested:
  `Exclude Go workspace files from Docker builds`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Exclude the workspace files from the build context

In `.dockerignore`, add a section after the "Build artifacts" section and
before "Environment", matching the existing style (a `#` heading, one entry
per line, a blank line between sections):

```
# Go workspace
go.work
go.work.sum
```

**Verify**: `grep -n "go.work" .dockerignore` → exactly two matches.

### Step 2: Exclude them from the compose watch sync

In `compose.yaml`, add two entries to the `ignore` list of the
`sync+restart` action, after `bin/`, with the same indentation:

```yaml
              - go.work
              - go.work.sum
```

**Verify**: `grep -n "go.work" compose.yaml` → exactly two matches, and
`git diff compose.yaml` shows only those two added lines.

### Step 3: Optional Docker evidence

If Docker is available, run `docker compose config --quiet`, then the builder
image build from the commands table in the background.

**Verify**: both exit 0. If Docker is not available, say so in your report
and skip this step.

## Test plan

No tests. This repository's rules forbid adding, running, or proposing tests
unless the maintainer explicitly asks. Verification is the grep checks and
the optional Docker build.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -c "go.work" .dockerignore` prints `2`
- [ ] `grep -c "go.work" compose.yaml` prints `2`
- [ ] `git diff --stat` lists only `.dockerignore` and `compose.yaml`
- [ ] `grep -c "\[" compose.yaml` prints the same number before and after your change (markers untouched)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- `.dockerignore` already contains a `go.work` entry, or the `ignore` list in
  `compose.yaml` does not match the excerpt.
- The Docker build fails for a reason unrelated to the workspace file. Report
  the error; do not edit the Dockerfile.
- The fix appears to require touching an out-of-scope file.

## Maintenance notes

- If a future workflow wants containers to build against a sibling checkout,
  it needs a bind mount and an explicit workspace file inside the image; this
  exclusion would have to be revisited then.
- The `rebuild` trigger watches `go.mod` but not `go.sum`. That was left
  alone; it is a separate, minor choice.
- On the maintainer's machine, local builds still use the workspace and
  therefore a newer `fphttp` than `go.mod` pins. Plan 015 addresses the pin.

Standards: Meet the solution standards in ~/.agents/AGENTS.md: idiomatic, beautiful, elegant, seamless, well-structured, dynamic, and future-proof, as explained in ~/.agents/CODING.md.
Guidance: Before implementing, read the repository's AGENTS.md and STYLE.md (if present in your checkout) and read `.dockerignore` and `compose.yaml` in full.
Review: Read every changed file from top to bottom for readability before declaring the plan done.
Evidence: Verify with command output and the relevant file contents. Do not add or run tests unless tests are explicitly requested.
Execution: Run every shell command that may take more than a few seconds, and every subagent task, in the background, and keep working on what does not depend on it until its completion notification arrives. Never sleep or poll.
