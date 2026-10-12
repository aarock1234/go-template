# Principles and language versions

[Back to the Go entrypoint](../SKILL.md). Read the sections matching the current change; follow crosslinks when the change crosses a boundary.

- [Overview](#go-overview)
- [Philosophy](#go-philosophy)
- [Readable functions and explicit dependencies](#go-function-boundaries)
- [Reading and adapting the local template](#go-template-adaptation)

<a id="go-overview"></a>

# Go Style Guide

Project-agnostic Go style guide. Favor idiomatic Go over clever abstractions.

Read the repository's `go.mod` before choosing language or library features. The local `~/Documents/Coding/go-template` currently declares Go 1.27.2; an existing project's declared baseline takes precedence. Changing that baseline is a separate migration.

| Feature used here | Required Go version |
| --- | --- |
| Range over integers, per-iteration loop variables, enhanced ServeMux patterns | 1.22 |
| Range over iterator functions, `iter.Seq`, `iter.Seq2` | 1.23 |
| JSON `omitzero` | 1.24 |
| `sync.WaitGroup.Go` | 1.25 |
| `errors.AsType`, `new(expr)`, signal cancellation causes | 1.26 |
| Methods with their own type parameters | 1.27 |

The earlier guide's Go 1.23 blanket minimum did not cover all its examples. Use the version annotations here and preserve compatibility rather than silently upgrading a repository. Check official release information when a task calls for an upgrade; this guide does not pin an evergreen “latest stable” version.


Sources: [reference 1](https://go.dev/doc/go1.24), [reference 2](https://go.dev/doc/go1.26), [reference 3](https://go.dev/doc/go1.27), [reference 4](https://pkg.go.dev/sync#WaitGroup.Go).

<a id="go-philosophy"></a>

## Philosophy

Write boring code. Prefer explicit over implicit. Optimize for reading, not writing. Design zero values to work without initialization. Keep functions small and interfaces smaller.


<a id="go-function-boundaries"></a>

## Readable functions and explicit dependencies

Extract a function when its name can express one meaningful operation, its inputs make the dependency clear, and its output or side effects have a coherent contract. Good boundaries often separate decoding from domain validation, resource acquisition from processing, or state transitions from transport responses. Those divisions let a reader follow the main operation without repeatedly reconstructing lower-level details.

Keep a short sequence together when splitting it would force readers to jump among trivial wrappers or pass a long list of temporary state. Function length is a signal to inspect responsibilities, not a universal limit. Prefer a clear loop over a chain of generic helpers when the loop makes ordering, cancellation, or early exit easier to see.

Use guard clauses for invalid input and failed operations. Keep the successful path visible, name variables for their role, and use a named configuration value when positional arguments would be hard to distinguish. Add an interface, option system, or generic abstraction when an actual variation needs it; a hypothetical future variation does not justify making today's code harder to read.

Read [functional options](structs.md#go-functional-options), [consumer interfaces](interfaces.md#go-consumer-interfaces), and [package layers](architecture.md#go-layers) when deciding whether a larger abstraction earns its cost. Preserve the repository's conventions while applying these decisions to the concrete operation.


Sources: [reference 1](https://go.dev/wiki/CodeReviewComments#indent-error-flow), [reference 2](https://go.dev/blog/when-generics).

<a id="go-template-adaptation"></a>

## Reading and adapting the local template

When available, use `~/Documents/Coding/go-template` as the concrete example for organization and coding style. Read the files related to the task before implementation; the template's current source, module versions, and repository instructions are more useful than assuming every copied fragment still matches it.

| Task | Template files to inspect | Decision to carry into the implementation |
| --- | --- | --- |
| Startup and shutdown | `go.mod`, `cmd/template/main.go` | Module baseline, composition root, listener outcome, resource lifetime |
| Concurrent work | `pkg/worker/worker.go` | Positive bounds, cancellation, joining, result order |
| HTTP calls | `pkg/client/`, `pkg/template/request.go` | Actual transport types, request deadlines, body ownership, redirect policy |
| Service boundaries | `pkg/service/service.go` | A consumer-owned contract with only the required methods |
| Retrying an operation | `pkg/retry/retry.go` | Retry eligibility, side-effect safety, total budget |
| Persistent state | `pkg/state/` | Platform locking, concurrency contract, durability requirement |

Preserve the template's useful structure and readable naming while evaluating behavior against the current task. Its specialized browser-fingerprinting client is not a required dependency for ordinary HTTP applications, and a reusable helper's existence does not establish that its error, ownership, or concurrency policy fits every caller.

If the template is unavailable, proceed from these references and the repository's established conventions. Do not create a replacement template or change the repository's architecture just to reproduce the template's file tree.


Sources: [reference 1](https://go.dev/doc/modules/layout).
