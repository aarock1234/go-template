---
name: go-style
description: Apply Go style when writing, editing, refactoring, or reviewing .go files or go.mod. Covers readable code, package layout, errors, interfaces, typed boundaries, concurrency ownership, context, slog, and application lifecycle. Read the topic references that match the change.
---

# Go style

Write explicit, readable, idiomatic Go with small functions, small consumer interfaces, and purposeful spacing. Design useful zero values where possible. Make resource ownership, cancellation, and error handling visible to the caller.

## How to navigate

1. Read the repository's instructions and relevant existing files. Check `go.mod`, toolchain, dependency versions, and build constraints before selecting language or library APIs.
2. Read [principles and versions](references/foundations.md) and [module conventions](references/modules.md) for every code change.
3. Open the relevant topic directly from the task table. Read its local contents list, then the matching sections and linked dependencies. Follow links beside the affected rule, returning to the table when the concern changes. Follow all rows touched by the change; leave unrelated topics unloaded.
4. Before finishing, revisit the applicable sections against the actual diff. Account for every acquired resource, goroutine, error, and domain constraint. Explain a deliberate exception when a convention cannot fit the code's contract.

Use the configured Go reference project as the primary concrete example when available; the conventional local location is `~/Documents/Coding/go-template`. Read the corresponding files before implementing: `cmd/template/main.go`, `pkg/service/service.go`, and packages such as `worker`, `retry`, `client`, `state`, `env`, and `log`. The template currently targets Go 1.27 and uses `fphttp` for specialized HTTP work. Read each implementation critically; a template is an organization and style reference, not proof that every example suits the current task. If unavailable, follow the repository's existing organization and these references.

## Find the guidance for this change

| When changing… | Read… |
| --- | --- |
| Imports, names, packages, spacing, comments | [Modules](references/modules.md) |
| Error contracts, wrapping, sentinels, close failures, panic | [Errors](references/errors.md) |
| Struct layout, constructors, options, embedding, zero values, receivers | [Structs](references/structs.md) |
| Generics, JSON decoding, optional fields, collections, constants | [Types and boundaries](references/types.md) |
| Dependency contracts or interface composition | [Interfaces](references/interfaces.md) |
| Workers, goroutines, channels, locks, errgroup, retries | [Concurrency](references/concurrency.md), then [context](references/context.md) |
| Ordinary HTTP clients, specialized transports, proxies, response bodies | [Client applicability and body lifetime](references/http.md#go-http-client), [errors](references/errors.md), and [context](references/context.md) |
| Project packages, layers, dependency wiring | [Architecture](references/architecture.md), then [lifecycle](references/lifecycle.md) when wiring a long-running process |
| Cancellation, deadlines, request-scoped values | [Context](references/context.md) |
| Log setup, attributes, error reporting, redaction | [Logging](references/logging.md) |
| Lazy iteration, sequences, collection traversal | [Iterators](references/iterators.md) |
| Server start/stop, process lifetime, cleanup, file state and persistence | [Lifecycle](references/lifecycle.md), [context](references/context.md), and [errors](references/errors.md) |
| Database queries, row iteration, transactions, and persistence | [Database work](references/data.md), then [errors](references/errors.md) and [context](references/context.md) |
| Dependencies, static analysis, formatting, modernizing APIs | [Workflow](references/workflow.md) |

## Scope and working constraints

- These are project conventions. Repository conventions and explicit user instructions determine how they apply; language correctness and public contracts still matter.
- Match the module's supported Go version. Read [version applicability](references/foundations.md) before using `omitzero`, `WaitGroup.Go`, `new(expr)`, `errors.AsType`, or generic methods.
- Favor clear names, focused files, readable literals, and logical whitespace over compressed code. Formatter compliance alone does not establish readability.
- Never add, write, run, propose, or plan tests unless the user explicitly requests them. Static analysis and formatting checks are separate; inspect Makefile targets and aggregate commands for indirect test execution.
- Dependency updates and source-rewriting commands require a reason within the current task. They are not routine read-only checks.

## Maintaining the guidance

The topic references contain the guidance, examples, and exceptions. Link directly to the relevant section when explaining a finding; distinguish a correctness issue from a contextual style preference. Add missing guidance to its owning topic rather than creating another project-level rulebook.

Existing section anchors preserve working links. When moving a section, update its incoming links or keep a compatible anchor.

[STYLE.md](STYLE.md) remains a compatibility index for older links. Detailed guidance lives only in the topic references.
