# Cancellation and context propagation

[Back to the Go entrypoint](../SKILL.md). Read the sections matching the current change; follow crosslinks when the change crosses a boundary.

- [Context Propagation](#go-context)

<a id="go-context"></a>

## Context Propagation

Put `context.Context` first in functions that need cancellation, deadlines, or request-scoped metadata, including I/O. Pass it through the call chain rather than replacing it with a fresh background context midway through an operation.

Pass context per call instead of storing it in a reusable service struct, so each call has its own cancellation scope. An adapter forced to implement a fixed external method signature may need a narrowly scoped context field; document that adapter's lifetime explicitly rather than applying the exception to ordinary services.

```go
// BAD: context stored in struct
type Service struct {
    ctx context.Context  // stale context, unclear lifecycle
}

// GOOD: context passed per-call
func (s *Service) Process(ctx context.Context, id string) error {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    return s.doWork(ctx, id)
}
```

Use `context.Background()` at a top-level owner such as `main`. Use `context.TODO()` as an explicit temporary placeholder while the correct owner is not yet available. Never pass nil.

Call each derived context's cancel function on every exit path; cancellation also releases timer and parent references. Values are for request metadata crossing API boundaries, not dependencies or optional function parameters. Starting a goroutine with a context does not make it stop automatically: blocking operations must observe cancellation and the owner must still join the work.

`ctx.Err()` supplies standard cancellation or deadline identity. `context.Cause(ctx)` supplies an owner-recorded reason on Go 1.20+, which may not match `context.Canceled`. Choose returned/wrapped errors according to the public contract. A canceled context does not prove an unrelated listener, storage, or cleanup failure arose from cancellation; preserve that failure instead of turning it into success merely because a cause exists.

A lifetime owner may use a fresh bounded context for cleanup that must outlive request cancellation; this is a deliberate cleanup exception, not a reason to detach ordinary work. See [server shutdown](lifecycle.md#go-shutdown) and [transaction release](data.md#go-transactions).

Source: [context causes](https://pkg.go.dev/context@go1.26.1#WithCancelCause).


Sources: [reference 1](https://pkg.go.dev/context), [reference 2](https://go.dev/wiki/CodeReviewComments#contexts).
