# Plan 014: Five small edge cases in the client, middleware, and ignore rules are closed

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 3112ea3..HEAD -- pkg/client/proxy.go pkg/client/cookie.go pkg/client/option.go pkg/middleware/writer.go .gitignore`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none (plan 001 edits `pkg/client/jar.go`, a different file)
- **Category**: bug
- **Planned at**: commit `3112ea3`, 2026-11-23

## Why this matters

Each of these is small, but each is a trap a user of the template would hit
without warning:

1. A proxy configured with a username and no password is sent without any
   credentials, so the proxy answers 407 and nothing explains why.
2. A proxy's password is redacted only when a `*Proxy` is formatted. A
   `Proxy` value logged with `slog` or `%v` prints the password.
3. A JSON cookie list containing `null` crashes the caller with a nil
   pointer panic instead of returning an error.
4. `.gitignore` covers `.env` but not `.env.local` or `.env.production`, so
   `git add .` commits those secrets.
5. The response-writer wrapper records "the response has started" before the
   real `WriteHeader` runs. If that call panics on an invalid status code,
   panic recovery believes a response exists and writes nothing, so the
   client receives an empty 200 instead of a 500.

## Current state

### 1 and 2 — `pkg/client/proxy.go:44-72`

```go
// Proxy holds a parsed proxy configuration.
type Proxy struct {
	Scheme   ProxyScheme
	Host     string
	Port     string
	Username string
	Password string
}

// URL returns the proxy as a URL suitable for Transport.Proxy, including
// its credentials.
func (p *Proxy) URL() *url.URL {
	u := &url.URL{
		Scheme: string(p.Scheme),
		Host:   net.JoinHostPort(p.Host, p.Port),
	}

	if p.Username != "" && p.Password != "" {
		u.User = url.UserPassword(p.Username, p.Password)
	}

	return u
}

// String returns the proxy URL with its password redacted, so a Proxy can
// be logged or formatted safely.
func (p *Proxy) String() string {
	return p.URL().Redacted()
}
```

`ParseProxy` (same file) only produces two or four fields, so it never
yields a username without a password; a hand-built `Proxy` can.
`Proxy` has exactly these two methods. Callers: `pkg/client/constructor.go:79`
(`c.proxy.URL()` on a `*Proxy`) and `pkg/client/client.go:114-122`.

The repository's style guide names `slog.LogValuer` as the way to keep
sensitive values out of logs.

### 3 — `pkg/client/cookie.go:14-24`

```go
func ParseCookies(s string) ([]*http.Cookie, error) {
	s = strings.TrimSpace(s)

	if strings.HasPrefix(s, "[") {
		var cookies []*http.Cookie
		if err := json.Unmarshal([]byte(s), &cookies); err != nil {
			return nil, fmt.Errorf("decode json cookies: %w", err)
		}

		return cookies, nil
	}
```

Its caller, `pkg/client/client.go:57-58`, reads `cookie.Name` for every
element.

### 4 — `.gitignore:30-31`

```
# env file
.env
```

`.dockerignore:10-13` already has the complete form:

```
# Environment
.env
.env.*
!.env.example
```

### 5 — `pkg/middleware/writer.go:13-21`

```go
// WriteHeader records the first status code before delegating.
func (w *responseWriter) WriteHeader(code int) {
	if !w.wrote {
		w.status = code
		w.wrote = true
	}

	w.ResponseWriter.WriteHeader(code)
}
```

`pkg/middleware/recover.go:37-39` writes the 500 only when `!rw.wrote`.
`net/http` panics in `WriteHeader` for a code outside 100–999, and treats
1xx codes as informational: more headers and a final status may follow.
`pkg/handler/handler.go:58` passes `appErr.Status` to the response
unvalidated, so a zero status reaches `WriteHeader(0)`.

### A doc claim to check — `pkg/client/option.go:18-19`

```go
// WithProxy routes requests through a copy of p. A nil p disables the
// proxy.
```

`pkg/client/constructor.go:78-80` sets `transport.Proxy` only when a proxy
is configured, so with nil the transport keeps whatever default
`github.com/aarock1234/fphttp/v2` gives it. If that default is
`ProxyFromEnvironment`, the comment is wrong.

### Conventions (from `AGENTS.md` and `STYLE.md`)

- Every nonempty struct literal is multiline with keyed fields.
- Do not mix pointer and value receivers on one type.
- Error strings are lowercase, wrapped with `fmt.Errorf("context: %w", err)`.
- Doc comments state the contract; keep them accurate.
- In `pkg/client`, `http` is an alias for `github.com/aarock1234/fphttp/v2`.

## Commands you will need

A local, git-ignored `go.work` may exist on the maintainer's machine. Prefix
Go commands with `GOWORK=off` so results match CI.

| Purpose      | Command                                                                                | Expected on success |
|--------------|----------------------------------------------------------------------------------------|---------------------|
| Build        | `GOWORK=off go build ./...`                                                            | exit 0, no output   |
| Vet          | `GOWORK=off go vet ./...`                                                              | exit 0, no output   |
| Format check | `gofmt -l pkg`                                                                         | no output           |
| Lint         | `GOWORK=off go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run` | exit 0, `0 issues.` |
| Find fphttp  | `GOWORK=off go list -m -f '{{.Dir}}' github.com/aarock1234/fphttp/v2`                  | prints the module directory |

Run lint once before changing anything to learn the baseline; run it in the
background.

## Scope

**In scope** (the only files you should modify):

- `pkg/client/proxy.go`
- `pkg/client/cookie.go`
- `pkg/client/option.go` (doc comment only, and only if Step 6 says so)
- `pkg/middleware/writer.go`
- `.gitignore`

**Out of scope** (do NOT touch, even though they look related):

- `pkg/client/jar.go` — plan 001 owns it.
- `pkg/client/constructor.go` — do not change how the transport's proxy is
  set; only the documentation is corrected.
- `pkg/handler/handler.go` and `pkg/apperr` — status validation is not added
  there; Step 5 makes an invalid status end as a logged 500.
- `pkg/middleware/recover.go`, `pkg/middleware/logger.go`.
- JSON tags on `Proxy`.
- Any test file. This repository forbids adding or running tests unless the
  maintainer asks for them.
- Uncommitted maintainer work in `README.md`, `AGENTS.md`, `STYLE.md`,
  `pkg/service/service.go`, and `.agents/skills/go-style/`. Do not stash,
  revert, commit, or reformat those files.

## Git workflow

- Branch: `advisor/014-client-and-middleware-edge-cases`
- One commit per step. Message style is a short imperative sentence, as in
  `git log`: "Use fphttp v2 profiles in the client". Example:
  `Send username-only proxy credentials`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Send a username even when there is no password

In `Proxy.URL`, set the user information whenever a username is present:
`url.UserPassword` when there is also a password, `url.User` when there is
not. Update the doc comment if it needs it.

**Verify**: `grep -n "url.User(" pkg/client/proxy.go` → one match;
`GOWORK=off go build ./pkg/client` → exit 0.

### Step 2: Redact however the proxy is formatted

Change both existing methods, `URL` and `String`, from pointer receivers to
value receivers (`func (p Proxy) ...`). A `*Proxy` still has both methods,
so no caller changes. Then add:

```go
// LogValue implements slog.LogValuer, so a logged Proxy never includes its
// password.
func (p Proxy) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
```

Import `log/slog`. Update the `String` comment to say that both `Proxy` and
`*Proxy` format redacted.

**Verify**: `grep -n "func (p \*Proxy)" pkg/client/proxy.go` → no matches;
`grep -c "func (p Proxy)" pkg/client/proxy.go` → prints `3`;
`GOWORK=off go build ./...` → exit 0.

### Step 3: Reject a null cookie

In the JSON branch of `ParseCookies`, after decoding, return an error when
the slice contains a nil element (`slices.Contains(cookies, nil)`), in the
style of the existing error, for example
`errors.New("decode json cookies: null cookie")`. Extend the function's doc
comment with one clause saying a null entry is an error.

**Verify**: `GOWORK=off go build ./pkg/client` → exit 0;
`grep -n "null" pkg/client/cookie.go` → at least two matches.

### Step 4: Ignore every local env file

In `.gitignore`, make the env section match `.dockerignore`:

```
# env files
.env
.env.*
!.env.example
```

**Verify**: `git check-ignore -q .env.local` → exit 0;
`git check-ignore -q .env.example` → exit 1;
`git ls-files .env.example` → prints `.env.example`.

### Step 5: Record the response only once it has really started

Rewrite `responseWriter.WriteHeader` in `pkg/middleware/writer.go` so it
delegates first and records afterwards, and only for a final status:

```go
// WriteHeader delegates, then records the first final status code. An
// informational 1xx code does not start the response, and a code that
// net/http rejects by panicking is never recorded, so panic recovery can
// still answer.
func (w *responseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)

	if !w.wrote && code >= http.StatusOK {
		w.status = code
		w.wrote = true
	}
}
```

Leave `Write`, `Status`, and `Unwrap` unchanged.

**Verify**: `GOWORK=off go build ./pkg/middleware` → exit 0.

### Step 6: Check, then correct, the `WithProxy` comment

Find the fphttp module directory with the "Find fphttp" command and search
it: `grep -rn "ProxyFromEnvironment" <dir> --include=*.go`.

- If the default transport created by `NewClient` uses
  `ProxyFromEnvironment`: change the `WithProxy` comment to say that a nil
  `p` removes the configured proxy, after which the transport follows the
  `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment variables.
- If it does not: leave the comment alone and say so in your report.

**Verify**: your report states which branch applied and quotes the fphttp
file and line you relied on.

### Step 7: Full verification

Run build, vet, format check, and lint from the commands table.

**Verify**: each command gives its expected result.

## Test plan

No tests. This repository's rules forbid adding, running, or proposing tests
unless the maintainer explicitly asks. Verification is the build, vet,
format, and lint commands, the `git check-ignore` checks, and a read of each
changed function.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `GOWORK=off go build ./...` and `GOWORK=off go vet ./...` exit 0
- [ ] `gofmt -l pkg` prints nothing
- [ ] Lint reports no issue that was not in the baseline
- [ ] `grep -n "func (p \*Proxy)" pkg/client/proxy.go` returns no matches
- [ ] `grep -n "LogValue" pkg/client/proxy.go` matches
- [ ] `git check-ignore -q .env.local` exits 0 and `git check-ignore -q .env.example` exits 1
- [ ] In `pkg/middleware/writer.go`, the delegate call precedes the assignment to `w.wrote` inside `WriteHeader`
- [ ] `git diff --stat` lists only in-scope files
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- Any excerpt above does not match the live code.
- Changing `Proxy`'s receivers breaks a caller outside `pkg/client`, or lint
  objects to copying the struct.
- `.env.example` would become ignored.
- Step 6 cannot determine the transport's default from the fphttp source.
- A verification fails twice after a reasonable fix attempt.

## Maintenance notes

- `Proxy` now uses value receivers. New methods on it should too.
- `json.Marshal` of a `Proxy` still includes the password; nothing in the
  repository marshals one. Add `json:"-"` if that ever changes.
- With Step 5, a handler that sets an invalid status ends as a recovered
  panic: a logged stack trace and a 500 response. The request log line then
  shows 500 rather than 0.
- Reviewers should check that the `Logger` middleware still reports the
  right status for an ordinary request: it reads `rw.Status()`, which is
  unchanged.

Standards: Meet the solution standards in ~/.agents/AGENTS.md: idiomatic, beautiful, elegant, seamless, well-structured, dynamic, and future-proof, as explained in ~/.agents/CODING.md.
Guidance: Before implementing, read the repository's AGENTS.md and STYLE.md (if present in your checkout), load the go-style skill (`.agents/skills/go-style/SKILL.md`), and read every in-scope file in full.
Review: Read every changed file from top to bottom for readability before declaring the plan done.
Evidence: Verify with build, vet, formatting checks, lint, command output, and the relevant file contents. Do not add or run tests unless tests are explicitly requested.
Execution: Run every shell command that may take more than a few seconds, and every subagent task, in the background, and keep working on what does not depend on it until its completion notification arrives. Never sleep or poll.
