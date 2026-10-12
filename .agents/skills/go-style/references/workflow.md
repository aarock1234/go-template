# Dependencies and static analysis

[Back to the Go entrypoint](../SKILL.md). Read the sections matching the current change; follow crosslinks when the change crosses a boundary.

- [Tools](#go-tools)
- [Dependency Management](#go-dependencies)
- [Static Analysis](#go-static-analysis)

<a id="go-quick-reference"></a>

## Quick Reference

Use the [entrypoint task table](../SKILL.md) to select guidance, then [tools](#go-tools) and [static analysis](#go-static-analysis) for authorized checks.


<a id="go-tools"></a>

### Tools

Choose checks relevant to the requested change and the project's configured tools. Type checking, static analysis, and formatting checks are separate from tests.

```bash
go vet ./...                 # static analysis
gofmt -l .                   # check formatting
go build -o bin/app ./cmd/app
```

The explicit user instruction supersedes the original automatic testing workflow: do not add, write, run, propose, or plan tests unless explicitly requested. This includes adding test dependencies or frameworks. Select a build only when producing or validating a compiled application is relevant.

`go mod tidy` changes dependency files; `go fix` rewrites source. Use them when dependency reconciliation or modernization is part of the task, then review the resulting diff. They are not read-only checks.


Sources: [reference 1](https://pkg.go.dev/cmd/vet), [reference 2](https://go.dev/doc/go1.26).

<a id="go-workflow"></a>

## Development Workflow

Follow the [repository versions and adaptation guidance](foundations.md#go-template-adaptation) before choosing [dependency](#go-dependencies) or [analysis](#go-static-analysis) commands.


<a id="go-dependencies"></a>

### Dependency Management

For an import-first dependency change, `go mod tidy` can reconcile required and unused modules after editing the imports. Use `go get module@version` when selecting a deliberate version; follow it with tidy when reconciliation is needed. Both commands mutate module files and may access the network. Review `go.mod` and `go.sum` changes as part of the requested change.

**Workflow:**

1. Add import to your `.go` file: `import "github.com/user/package"`
2. Run `go mod tidy` to fetch and add to `go.mod`

```bash
# PREFERRED: Import in code, then tidy
go mod tidy

# Use go get for specific versions
go get github.com/user/package@v1.2.3

# Select an intended upgrade explicitly, then review the diff
go get github.com/user/package@v1.3.0
```

**Why prefer `go mod tidy`:**

- Automatically manages both additions and removals
- Ensures `go.mod` and `go.sum` are in sync
- Removes unused dependencies
- Cleaner workflow: write code first, manage deps second
- Avoids separately managing each import when no specific version choice is needed

The earlier blanket upgrade example was broader than ordinary dependency maintenance. Choose versions and upgrade scope deliberately rather than upgrading the whole graph during unrelated work. Tidy can add dependencies needed across supported build configurations; a large diff deserves explanation, not automatic acceptance.


Sources: [reference 1](https://go.dev/ref/mod#go-mod-tidy), [reference 2](https://go.dev/ref/mod#go-get).

<a id="go-static-analysis"></a>

### Static Analysis

Use `go vet` for selected suspicious constructs such as incorrect format arguments, lock copying, lost cancellation, and malformed tags. Vet loads and analyzes packages and can require build/type information; do not promise it is always faster than a build. Its heuristics provide useful diagnostics, not proof of program correctness.

```bash
go vet ./...                 # all packages
go vet ./internal/services   # specific package
gofmt -l .                   # check formatting
go vet ./... && gofmt -l .   # combine checks
```

**Common issues `go vet` catches:**

- Printf format string mismatches
- Unreachable code
- Incorrect use of sync primitives
- Struct tags validation

**Additional analysis (requires explicit opt-in):**

- Shadow variables: `go vet -vettool=$(which shadow) ./...` or enable in `golangci-lint`

**When to use:**

- During development for quick validation
- In pre-commit hooks
- In CI/CD as a separate static-analysis step

If the project includes a `golangci-lint` configuration, use its configured lint workflow rather than repeating the same individual analyzers. Read the installed version and configuration: Golangci-lint v2 separates linters from formatters, so a lint run alone does not establish that configured formatting checks ran. The template has a v2 `formatters` section for gofmt/goimports; select the project's supported formatting check separately.

Use `go fix` when modernization is part of the requested task, with the intended module baseline, and review its edits. Go 1.26 introduced the revamped modernizers; older toolchains do not provide the same behavior. Do not run source-rewriting or dependency-changing tools as if they were read-only checks. Static analysis does not authorize adding or running tests.


Sources: [reference 1](https://pkg.go.dev/cmd/vet), [reference 2](https://golangci-lint.run/docs/welcome/quick-start/), [reference 3](https://go.dev/doc/go1.26).
