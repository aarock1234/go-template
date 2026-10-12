# Error contracts and resource cleanup

[Back to the Go entrypoint](../SKILL.md). Read the sections matching the current change; follow crosslinks when the change crosses a boundary.

- [Basic Pattern](#go-error-propagation)
- [Deferred Close Errors](#go-close-errors)
- [Sentinel Errors](#go-sentinel-errors)
- [Custom Error Types](#go-error-types)
- [When to Panic](#go-panic)

<a id="go-errors"></a>

## Error Handling

Start with [propagation](#go-error-propagation), then [cleanup](#go-close-errors), [inspection](#go-error-types), or [panic boundaries](#go-panic) as needed.


<a id="go-error-propagation"></a>

### Basic Pattern

Return errors as the last value and check them immediately. Add useful operation context with `fmt.Errorf`. Use `%w` when callers should be able to inspect the underlying failure with `errors.Is` or `errors.As`/`errors.AsType`.

At a public abstraction boundary, decide whether an implementation error is part of the supported API. Translate it into a domain error when callers should not depend on a specific database or transport error. Wrapping is deliberate API exposure, not a required spelling for every error.

```go
func (s *Service) Get(ctx context.Context, id string) (*Item, error) {
    item, err := s.repo.Get(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get item %s: %w", id, err)
    }

    return item, nil
}
```

Error messages begin lowercase unless a proper noun or initialism requires otherwise; omit trailing punctuation and add useful context. Give each failure one owner for logging. Logging an error and returning that same failure usually creates duplicate noise up the call chain. Handling individual failures and returning a distinct batch summary is a separate, explicitly documented policy.

```go
// BAD: logs and returns; caller will likely log again
if err != nil {
    s.logger.Error("failed to get item", "error", err)
    return nil, fmt.Errorf("get item: %w", err)
}

// GOOD: propagate with context, let the caller decide
if err != nil {
    return nil, fmt.Errorf("get item %s: %w", id, err)
}
```

Preserve independent failures during [shutdown](lifecycle.md#go-shutdown); the process context being canceled does not make a listener or cleanup error successful.


Sources: [reference 1](https://go.dev/blog/go1.13-errors#whether-to-wrap), [reference 2](https://go.dev/wiki/CodeReviewComments#error-strings).

<a id="go-close-errors"></a>

### Deferred Close Errors

The acquiring scope owns cleanup unless it explicitly transfers ownership. For read-only response bodies or files, a close failure can often be deliberately discarded once the read outcome is known. A network connection's completion semantics depend on the protocol; do not classify every connection as read-only cleanup. Use `_ =` to communicate an intentional discard:

```go
// GOOD: explicit discard for read-only cleanup
defer func() { _ = resp.Body.Close() }()
defer func() { _ = f.Close() }()

// A configured error checker may flag an unchecked cleanup result.
defer resp.Body.Close()
```

For writable resources, propagate failures from the operation that completes the write: flush a buffered writer, finish a compressor, commit a transaction, and close the file as required by its API. Closing a file does not flush a separate `bufio.Writer`, and close alone does not establish crash durability. Preserve the primary failure if cleanup also fails.

The former example only logged a writable close error and could let its caller report success. This unbuffered file example returns the failure instead; its named result is intentional so deferred cleanup can amend the outcome:

```go
func writeFile(path string, data []byte) (err error) {
    file, err := os.Create(path)
    if err != nil {
        return fmt.Errorf("create output: %w", err)
    }

    defer func() {
        if closeErr := file.Close(); closeErr != nil {
            err = errors.Join(err, fmt.Errorf("close output: %w", closeErr))
        }
    }()

    if _, err := file.Write(data); err != nil {
        return fmt.Errorf("write output: %w", err)
    }

    return nil
}
```

A function returning an HTTP response transfers body ownership to its caller. A wrapper around a resource must state whether closing the wrapper also closes the underlying resource. Inside loops, place per-item acquisition and deferred cleanup in a small function when keeping all resources open until the outer function returns would be excessive.

For database capabilities, use the driver-specific [transaction](data.md#go-transactions) and [row cursor](data.md#go-query-rows) lifetime contracts.


Sources: [reference 1](https://pkg.go.dev/bufio#Writer.Flush), [reference 2](https://pkg.go.dev/os#File.Close), [reference 3](https://go.dev/doc/database/execute-transactions).

<a id="go-sentinel-errors"></a>

### Sentinel Errors

For expected error conditions. Check with `errors.Is`.

```go
var (
    ErrNotFound     = errors.New("not found")
    ErrUnauthorized = errors.New("unauthorized")
    ErrInvalidInput = errors.New("invalid input")
)

if errors.Is(err, ErrNotFound) {
    // handle not found
}
```


<a id="go-error-types"></a>

### Custom Error Types

For errors that carry additional data, use a custom error type. On Go 1.26+, inspect it with `errors.AsType`. On an older supported baseline, declare a target and use `errors.As(err, &target)`.

```go
type ValidationError struct {
    Field string
    Reason string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed: %s: %s", e.Field, e.Reason)
}

// Usage
if valErr, ok := errors.AsType[*ValidationError](err); ok {
    log.Error("validation failed", "field", valErr.Field)
}
```

Keep the public return type `error`. A nil `*ValidationError` stored in an `error` interface is still a non-nil interface; successful paths should return `nil` explicitly.


Sources: [reference 1](https://go.dev/doc/go1.26#errors), [reference 2](https://go.dev/doc/faq#nil_error).

<a id="go-panic"></a>

### When to Panic

Reserve panic for violated internal invariants or setup that truly cannot be represented as a recoverable error. Return ordinary configuration, network, and storage failures from constructors or `run`, including failures formerly hidden inside `init`. Library operational errors belong in return values.

An `init` panic is appropriate only when process-wide initialization is deliberately unavoidable and failure leaves no usable program; it is not a reason to move fallible application setup into `init`. A fixed program literal compiled during initialization makes the invariant visible; input-dependent patterns should use `regexp.Compile` and return its error.

```go
// A malformed fixed program literal is an internal programming error.
var itemIDPattern = regexp.MustCompile(`^[a-z0-9]{16}$`)

// status has already passed boundary validation.
switch status {
case StatusActive, StatusPending, StatusCompleted:
    // ...
default:
    panic(fmt.Sprintf("unhandled status: %v", status))
}
```

Validate externally supplied enums before relying on an exhaustive switch; see [constants](types.md#go-constants).

Recover only at a boundary that can establish a defined failure outcome after unwinding; never recover an operational panic and report success. Recovery applies to the same goroutine: a caller's deferred recovery does not catch a panic in a launched worker. Keep `WaitGroup.Go`'s no-panic requirement explicit instead of adding blanket recovery to every worker.

Source: [panic and recovery scope](https://go.dev/ref/spec#Handling_panics).


Sources: [reference 1](https://go.dev/wiki/CodeReviewComments#dont-panic).
