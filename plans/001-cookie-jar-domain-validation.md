# Plan 001: The cookie jar only stores a cookie for a domain its sender may set

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 3112ea3..HEAD -- pkg/client/jar.go go.mod go.sum`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: security
- **Planned at**: commit `3112ea3`, 2026-11-23

## Why this matters

`pkg/client` installs its own `CookieJar` on every client. The jar files a
cookie under whatever `Domain` attribute the response carries and never checks
that attribute against the host that sent it. Any host the client touches —
a redirect hop, a third-party endpoint — can therefore plant or overwrite
cookies for an unrelated domain, or for an entire top-level domain, and the
client will send them there. In a scraper that talks to several targets this
is cross-site cookie fixation. Browsers and `net/http/cookiejar` reject such
cookies; after this plan the template's jar does too.

## Current state

- `pkg/client/jar.go` — the whole jar: `SetCookies`, `Cookies`, `Clear`, and
  the helpers `jarKey`, `storedCookie`, `expired`, `domainMatch`.
- `pkg/client/client.go:49-72` — `SetCookieString` seeds cookies by setting
  `cookie.Domain = domain` and calling the jar with a URL whose host is the
  same `domain`. This must keep working.
- `pkg/client/constructor.go:32` — `c.http.Jar = c.jar`; the HTTP client hands
  response cookies to the jar unfiltered.

The type documentation, `pkg/client/jar.go:14-16`:

```go
// CookieJar stores cookies by domain. A cookie with a Domain attribute
// matches that domain and its subdomains; a cookie without one matches only
// the host that set it. Path and Secure attributes are not enforced.
```

The storing loop, `pkg/client/jar.go:41-43`:

```go
	for _, cookie := range cookies {
		key := jarKey(u, cookie)
		stored := j.cookies[key]
```

The key function, `pkg/client/jar.go:107-114`:

```go
// jarKey returns the key that stores a cookie received from u.
func jarKey(u *url.URL, cookie *http.Cookie) string {
	if cookie.Domain == "" {
		return strings.ToLower(u.Hostname())
	}

	return "." + strings.ToLower(strings.TrimPrefix(cookie.Domain, "."))
}
```

The lookup, `pkg/client/jar.go:136-145`:

```go
// domainMatch reports whether host matches a jar key. A leading-dot key
// matches the domain itself and any of its subdomains.
func domainMatch(host, key string) bool {
	domain, isDomain := strings.CutPrefix(key, ".")
	if !isDomain {
		return host == key
	}

	return host == domain || strings.HasSuffix(host, key)
}
```

`go.mod:48` already lists `golang.org/x/net v0.61.0 // indirect`. Its
`publicsuffix` package provides the public-suffix list.

Conventions that apply (from the repository's `AGENTS.md` and `STYLE.md`):

- Every nonempty struct literal is multiline with keyed fields.
- Small focused functions, purposeful blank lines, a doc comment on every
  function that states its contract (see the existing comments in `jar.go`).
- Imports are grouped: standard library, third party, then
  `github.com/aarock1234/go-template/...`. In this file `http` is an alias for
  `github.com/aarock1234/fphttp/v2`; keep that.

## Commands you will need

A local, git-ignored `go.work` may exist on the maintainer's machine. Prefix
Go commands with `GOWORK=off` so results match CI.

| Purpose      | Command                                                                                     | Expected on success        |
|--------------|---------------------------------------------------------------------------------------------|----------------------------|
| Build        | `GOWORK=off go build ./...`                                                                 | exit 0, no output          |
| Vet          | `GOWORK=off go vet ./...`                                                                   | exit 0, no output          |
| Format check | `gofmt -l pkg/client`                                                                       | no output                  |
| Tidy check   | `GOWORK=off go mod tidy -diff`                                                              | exit 0, no output          |
| Lint         | `GOWORK=off go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run`      | exit 0, `0 issues.`        |

Run the lint command once before changing anything to learn the baseline. If
the baseline already reports issues, record them and require only that you add
none. Lint can take minutes the first time: run it in the background and keep
working.

## Suggested executor toolkit

- Load the `go-style` skill if available (a copy lives at
  `.agents/skills/go-style/SKILL.md`).
- RFC 6265 section 5.3, steps 5 and 6, is the rule being implemented.

## Scope

**In scope** (the only files you should modify):

- `pkg/client/jar.go`
- `go.mod`, `go.sum` (only the change `go mod tidy` makes when
  `golang.org/x/net` becomes a direct requirement)

**Out of scope** (do NOT touch, even though they look related):

- `pkg/client/client.go` — `SetCookieString`, `Cookies`, `FindCookie` keep
  their behavior and signatures.
- `Path` and `Secure` enforcement — the type documents that they are not
  enforced; that is a separate decision.
- Any test file. This repository forbids adding or running tests unless the
  maintainer asks for them.
- Uncommitted maintainer work in `README.md`, `AGENTS.md`, `STYLE.md`,
  `pkg/service/service.go`, and `.agents/skills/go-style/`. Do not stash,
  revert, commit, or reformat those files.

## Git workflow

- Branch: `advisor/001-cookie-jar-domain-validation`
- One commit. Message style is a short imperative sentence, as in
  `git log`: "Use fphttp v2 profiles in the client", "Address review findings".
  Suggested: `Validate cookie domains in the client jar`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Replace `jarKey` with a function that can refuse a cookie

In `pkg/client/jar.go`, replace `jarKey` with a function that returns the key
and whether the cookie may be stored. The rule:

1. `host` is `strings.ToLower(u.Hostname())`. `domain` is the cookie's
   `Domain`, lowercased, without a leading dot.
2. An empty `domain` is a host-only cookie: key `host`, accepted.
3. When `host` is an IP address (use `net/netip`: `netip.ParseAddr(host)`
   succeeds), a domain equal to `host` is host-only (key `host`, accepted);
   any other domain is refused.
4. When `domain` is a public suffix (`publicsuffix.PublicSuffix(domain)`
   returns `domain` itself), a domain equal to `host` is host-only (key
   `host`, accepted); any other host is refused. This stops a cookie for
   `com` or `co.uk`.
5. Otherwise the cookie is accepted, with key `"." + domain`, only when
   `host == domain` or `host` ends with `"." + domain`. Anything else is
   refused.

Target shape (names are suggestions; keep the doc comment accurate):

```go
// cookieKey returns the key that stores a cookie received from u, and
// whether u may set it. A Domain attribute must name u's host or one of its
// parent domains, and must not be a public suffix such as "com"; a host may
// still set a cookie for itself.
func cookieKey(u *url.URL, cookie *http.Cookie) (key string, ok bool) {
```

Split the IP and public-suffix checks into small helpers if the function
stops reading top to bottom.

Import `net/netip` and `golang.org/x/net/publicsuffix`.

**Verify**: `grep -n "func jarKey" pkg/client/jar.go` → no matches.

### Step 2: Skip refused cookies in `SetCookies`

In the loop at `pkg/client/jar.go:41`, call the new function and `continue`
when it refuses the cookie. Keep the rest of the loop unchanged.

**Verify**: `GOWORK=off go build ./pkg/client` → exit 0.

### Step 3: Update the documentation

- `CookieJar` type comment (`jar.go:14-16`): add that a cookie whose `Domain`
  attribute does not cover the host that set it, or names a public suffix, is
  ignored.
- `SetCookies` comment (`jar.go:28-30`): mention that such cookies are
  skipped.

**Verify**: `grep -n "public suffix" pkg/client/jar.go` → at least two matches.

### Step 4: Make the new requirement direct

Run `GOWORK=off go mod tidy`. `golang.org/x/net` moves from the indirect
block of `go.mod` to the direct block; its version must not change.

**Verify**: `GOWORK=off go mod tidy -diff` → exit 0, no output, and
`git diff go.mod` shows only `golang.org/x/net v0.61.0` moving between
blocks.

### Step 5: Full verification

Run build, vet, format check, and lint from the commands table.

**Verify**: each command gives its expected result.

## Test plan

No tests. This repository's rules forbid adding, running, or proposing tests
unless the maintainer explicitly asks. Verification is the build, vet,
format, tidy, and lint commands above plus a top-to-bottom read of
`pkg/client/jar.go`.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `GOWORK=off go build ./...` exits 0
- [ ] `GOWORK=off go vet ./...` exits 0
- [ ] `gofmt -l pkg/client` prints nothing
- [ ] `GOWORK=off go mod tidy -diff` exits 0 with no output
- [ ] Lint reports no issue that was not in the baseline
- [ ] `grep -n "publicsuffix" pkg/client/jar.go` matches
- [ ] `grep -n "func jarKey" pkg/client/jar.go` returns no matches
- [ ] `git status --short` lists no file outside the in-scope list that was not already modified before you started
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The code in `pkg/client/jar.go` does not match the excerpts above.
- `go mod tidy` changes the version of any module, or adds a module other
  than making `golang.org/x/net` direct.
- `http.Cookie` (from `github.com/aarock1234/fphttp/v2`) has no `Domain`
  field, or `u.Hostname()` is unavailable.
- Making `SetCookieString` keep working appears to require editing
  `pkg/client/client.go`.
- A verification fails twice after a reasonable fix attempt.

## Maintenance notes

- `SetCookieString("example.com", ...)` still stores a domain cookie because
  the URL host equals the cookie domain. Calling it with a public suffix now
  stores a host-only cookie instead of a suffix-wide one.
- Reviewers should read the IP-address branch and the public-suffix branch
  closely; those are the two places a mistake silently drops legitimate
  cookies.
- Deferred: `Secure` and `Path` enforcement. Secure cookies are still sent
  over plain HTTP, as the type comment says.

Standards: Meet the solution standards in ~/.agents/AGENTS.md: idiomatic, beautiful, elegant, seamless, well-structured, dynamic, and future-proof, as explained in ~/.agents/CODING.md.
Guidance: Before implementing, read the repository's AGENTS.md and STYLE.md (if present in your checkout), load the go-style skill (`.agents/skills/go-style/SKILL.md`), and read `pkg/client/jar.go` in full.
Review: Read every changed file from top to bottom for readability before declaring the plan done.
Evidence: Verify with build, vet, formatting checks, lint, command output, and the relevant file contents. Do not add or run tests unless tests are explicitly requested.
Execution: Run every shell command that may take more than a few seconds, and every subagent task, in the background, and keep working on what does not depend on it until its completion notification arrives. Never sleep or poll.
