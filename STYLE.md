# Readable Go

## Why

Code is a document people maintain. Optimize for understanding an operation, not fewer lines or tokens. Use descriptive names, visible decisions, small focused functions, and purposeful whitespace so readers can follow the workflow continuously.

## Apply the guidance

Expand every nonempty struct declaration, struct literal, and keyed map literal, one field or entry per line. Use keyed struct fields, even for one field, nested arguments, or a returned value.

```go
type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

var labels = map[string]string{
	"status": "ready",
}
```

Empty values such as `Service{}` or `map[string]string{}` may stay compact. Collections of simple values are a separate layout decision. Preserve gofmt alignment and do not hand-reformat generated or vendored source; document genuine compatibility exceptions.

Separate I/O, decoding and boundary validation, domain decisions, orchestration, and presentation when ownership or reasons to change differ. Keep a short related workflow together rather than creating trivial wrappers or micro-files. Give each domain shape one authority and keep it with its capability. Define small interfaces where they are consumed; do not collect unrelated types or utilities in generic packages.

## Concrete examples and guidance

- `cmd/template/main.go`: process configuration, dependency construction, cancellation, and cleanup.
- `pkg/service/service.go`: a consumer-owned repository contract and expanded constructor result.
- `pkg/respond/respond.go`: response encoding and presentation, separate from domain decisions.
- `pkg/env/` and `pkg/state/`: configuration and persistence boundaries.

Read the relevant implementation before copying it; templates do not establish that a dependency or architecture fits every project. Preserve the repository's toolchain and setup markers.

[AGENTS.md](AGENTS.md) contains repository instructions. The local [go-style skill](.agents/skills/go-style/SKILL.md) owns detailed Go guidance, especially [record layout](.agents/skills/go-style/references/structs.md#go-struct-literals) and [responsibility boundaries](.agents/skills/go-style/references/architecture.md#go-layers).

## Review

Run gofmt on changed Go files, then read the actual result, including nested values and returns. Formatting cannot assess meaningful names, resource ownership, or responsibility boundaries. Check that each error, acquired resource, and started goroutine has an explicit outcome and owner.

Never add or run tests without an explicit request. Formatting and static analysis are separate; inspect Makefile targets before using aggregate commands.
