# Plan 013: Every GitHub Action runs from a pinned commit

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 3112ea3..HEAD -- .github`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: security
- **Planned at**: commit `3112ea3`, 2026-11-23

## Why this matters

The workflows reference actions by major-version tag, such as `@v3`. A tag
can be moved to different code at any time by whoever controls the action's
repository. The release workflow runs a third-party action in a job that
holds a token with `contents: write`, so a moved tag would run new code with
write access to the repository on the next tag push — in this repository and
in every project scaffolded from it, since the workflows are copied.
Referencing each action by full commit SHA makes what runs immutable, and a
Dependabot configuration keeps the pins current so they do not rot.

## Current state

- `.github/workflows/ci.yaml` (whole file, 35 lines):

  ```yaml
  name: CI

  on:
    push:
      branches: [main]
    pull_request:
      branches: [main]
    workflow_call:

  permissions:
    contents: read

  jobs:
    check:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v7

        - uses: actions/setup-go@v7
          with:
            go-version-file: go.mod

        - name: Tidy
          run: go mod tidy -diff

        - name: Lint
          uses: golangci/golangci-lint-action@v9
          with:
            version: v2.14.0 # keep in sync with GOLANGCI_LINT_VERSION in the Makefile

        - name: Build
          run: make build

        - name: Test
          run: make test
  ```

- `.github/workflows/release.yaml:12-23`:

  ```yaml
    release:
      name: Publish GitHub release
      needs: check
      runs-on: ubuntu-latest
      permissions:
        contents: write

      steps:
        - name: Create GitHub Release
          uses: softprops/action-gh-release@v3
          with:
            generate_release_notes: true
  ```

- There is no `.github/dependabot.yml`.

- No step after checkout pushes to the repository or otherwise needs the git
  credentials that `actions/checkout` stores by default.

- YAML in this repository uses two-space indentation and double quotes only
  where needed.

## Commands you will need

| Purpose             | Command                                                              | Expected on success                        |
|---------------------|----------------------------------------------------------------------|--------------------------------------------|
| Resolve a tag       | `git ls-remote https://github.com/<owner>/<repo> 'refs/tags/v<N>*'`  | lists tags with their commit SHAs          |
| No tag references   | `grep -rnE 'uses: [^.].*@v[0-9]' .github/workflows`                  | no matches                                 |

`git ls-remote` needs network access; run it in the background. When a tag
is annotated, `ls-remote` prints two lines for it: `refs/tags/vX.Y.Z` and
`refs/tags/vX.Y.Z^{}`. The SHA on the `^{}` line is the commit; use that one.

## Scope

**In scope** (the only files you should modify):

- `.github/workflows/ci.yaml`
- `.github/workflows/release.yaml`
- `.github/dependabot.yml` (create)

**Out of scope** (do NOT touch, even though they look related):

- The `uses: ./.github/workflows/ci.yaml` line in `release.yaml` — a local
  reusable workflow is not pinned by SHA.
- The `version: v2.14.0` input of the lint action and its comment; it must
  stay in sync with the Makefile.
- The steps themselves: do not add, remove, or reorder jobs or steps, and do
  not change which major version of an action is used.
- Container image tags in `Dockerfile` and `compose.yaml`.
- Uncommitted maintainer work in `README.md`, `AGENTS.md`, `STYLE.md`,
  `pkg/service/service.go`, and `.agents/skills/go-style/`. Do not stash,
  revert, commit, or reformat those files.

## Git workflow

- Branch: `advisor/013-pin-github-actions`
- One commit. Message style is a short imperative sentence, as in `git log`:
  "Use fphttp v2 profiles in the client". Suggested:
  `Pin GitHub Actions to commit SHAs`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Resolve each action's current release to a commit

For each of the four actions, list the tags within the major version the
workflow uses today and pick the highest full version (`vX.Y.Z`):

| Action                           | Major in use |
|----------------------------------|--------------|
| `actions/checkout`               | `v7`         |
| `actions/setup-go`               | `v7`         |
| `golangci/golangci-lint-action`  | `v9`         |
| `softprops/action-gh-release`    | `v3`         |

Record, for each, the full version and its 40-character commit SHA.

**Verify**: you have four `(version, SHA)` pairs, each SHA exactly 40
hexadecimal characters, each taken from the `^{}` line when one exists.

### Step 2: Replace the tag references

In both workflow files, replace each `@v<N>` with `@<SHA>` followed by a
comment naming the full version, for example:

```yaml
      - uses: actions/checkout@<40-character SHA> # v7.0.1
```

Dependabot reads that trailing comment and updates it together with the SHA.

**Verify**: `grep -rnE 'uses: [^.].*@v[0-9]' .github/workflows` → no
matches, and `grep -rnE 'uses: [^.].*@[0-9a-f]{40} # v[0-9]+\.[0-9]+\.[0-9]+' .github/workflows`
→ four matches.

### Step 3: Do not leave git credentials in the workspace

In `ci.yaml`, give the checkout step an input so the token is not persisted
for later steps:

```yaml
      - uses: actions/checkout@<SHA> # v7.x.y
        with:
          persist-credentials: false
```

**Verify**: `grep -n "persist-credentials: false" .github/workflows/ci.yaml`
→ one match.

### Step 4: Keep the pins current

Create `.github/dependabot.yml`:

```yaml
version: 2

updates:
  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: weekly
```

**Verify**: the file exists and `grep -c "github-actions" .github/dependabot.yml`
prints `1`.

## Test plan

No tests. This repository's rules forbid adding, running, or proposing tests
unless the maintainer explicitly asks. Verification is the grep checks; the
workflows themselves are exercised by GitHub on the next push, which is
outside this plan.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -rnE 'uses: [^.].*@v[0-9]' .github/workflows` returns no matches
- [ ] Four `uses:` lines carry a 40-character SHA and a `# vX.Y.Z` comment
- [ ] `grep -c "persist-credentials: false" .github/workflows/ci.yaml` prints `1`
- [ ] `.github/dependabot.yml` exists with the `github-actions` ecosystem
- [ ] `git diff --stat` lists only the three in-scope files
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- `git ls-remote` cannot reach GitHub. Do not guess or reuse a SHA from
  memory; a wrong SHA breaks CI or pins unknown code.
- An action has no tag within the major version listed above.
- The workflow files do not match the excerpts.
- A step in either workflow turns out to need persisted git credentials.

## Maintenance notes

- Reviewers should spot-check each SHA against the action's release page
  before merging; that is the whole point of the pin.
- Dependabot opens pull requests for new action releases. Merging them is
  now the way actions get updated; nothing floats.
- Scaffolded projects inherit `dependabot.yml`. A project not hosted on
  GitHub can delete the `.github` directory entirely.
- Deferred: pinning container images by digest. It was judged more
  maintenance than it is worth for a template's development stack.

Standards: Meet the solution standards in ~/.agents/AGENTS.md: idiomatic, beautiful, elegant, seamless, well-structured, dynamic, and future-proof, as explained in ~/.agents/CODING.md.
Guidance: Before implementing, read the repository's AGENTS.md and STYLE.md (if present in your checkout), and read both workflow files in full.
Review: Read every changed file from top to bottom for readability before declaring the plan done.
Evidence: Verify with command output and the relevant file contents. Do not add or run tests unless tests are explicitly requested.
Execution: Run every shell command that may take more than a few seconds, and every subagent task, in the background, and keep working on what does not depend on it until its completion notification arrives. Never sleep or poll.
