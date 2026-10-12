# Structs, constructors, and receivers

[Back to the Go entrypoint](../SKILL.md). Read the sections matching the current change; follow crosslinks when the change crosses a boundary.

- [Struct Literal Formatting](#go-struct-literals)
- [Constructor Pattern](#go-constructors)
- [Functional Options Pattern](#go-functional-options)
- [Struct Embedding](#go-embedding)
- [Zero Values](#go-zero-values)
- [Pointer vs Value Receivers](#go-receivers)
- [Ownership of mutable inputs and returned values](#go-mutable-input-ownership)

<a id="go-structs"></a>

## Struct Patterns

Choose [construction](#go-constructors), [options](#go-functional-options), [receiver behavior](#go-receivers), and [mutable ownership](#go-mutable-input-ownership) from the actual type contract.


<a id="go-struct-literals"></a>

### Struct Literal Formatting

Always use multi-line format for struct literals with two or more fields. Single-line is only acceptable for one field.

```go
// GOOD: single field, inline is fine
item := Item{ID: "abc"}

// GOOD: multiple fields, one per line
item := Item{
    ID:   "abc",
    Name: "example",
}

// BAD: multiple fields crammed on one line
item := Item{ID: "abc", Name: "example"}
```


<a id="go-constructors"></a>

### Constructor Pattern

Name constructors `New` as the package name already provides context. Callers read `service.New(...)`, not `service.NewService(...)`. This applies to the primary type in a package.

```go
type Service struct {
    repo   Repository
    logger *slog.Logger
    timeout time.Duration
}

func New(repo Repository, logger *slog.Logger) *Service {
    return &Service{
        repo:    repo,
        logger:  logger,
        timeout: 10 * time.Second,
    }
}
```


<a id="go-functional-options"></a>

### Functional Options Pattern

Use functional options when several independent optional settings make a constructor easier to evolve and read. Keep required dependencies explicit. A small configuration struct is often clearer for a small, cohesive group of settings; options are a project preference for complex configuration, not a requirement for every constructor.

Apply options, validate their combination once, and return a configuration error when invalid input could produce a panic, unlimited concurrency, or an unusable service. State whether mutable option values are copied or retained.

```go
type Option func(*Service)

func WithTimeout(d time.Duration) Option {
    return func(s *Service) { s.timeout = d }
}

func New(repo Repository, opts ...Option) (*Service, error) {
    s := &Service{
        repo:    repo,
        timeout: 10 * time.Second,
    }
    for _, opt := range opts {
        opt(s)
    }

    if s.timeout <= 0 {
        return nil, errors.New("timeout must be positive")
    }

    return s, nil
}

// Usage
svc, err := New(repo, WithTimeout(5*time.Second))
if err != nil {
    return fmt.Errorf("create service: %w", err)
}
```


<a id="go-embedding"></a>

### Struct Embedding

Embedding can promote deliberately shared behavior across implementations. Prefer a named field and explicit delegation when promotion would accidentally expand the containing type's public API. Embedding is composition, not inheritance; method shadowing changes which promoted method callers see.

```go
type BaseProcessor struct {
    logger *slog.Logger
    config Config
}

func (b *BaseProcessor) Validate(input Input) error {
    if input.ID == "" {
        return errors.New("input id required")
    }

    return nil
}

type PDFProcessor struct {
    BaseProcessor  // promotes Validate method
    pdfConfig PDFConfig
}

func (p *PDFProcessor) Process(input Input) error {
    if err := p.Validate(input); err != nil {
        return err
    }
    // PDF-specific logic goes here.

    return nil
}

// Note: if PDFProcessor later defines its own Validate method,
// it silently shadows BaseProcessor.Validate.
```


Sources: [reference 1](https://go.dev/doc/effective_go#embedding).

<a id="go-zero-values"></a>

### Zero Values

Design structs to be usable without initialization when possible.

```go
// GOOD: zero value works
var cache Cache
cache.Set("key", "value")  // works even if not initialized

// Implementation
type Cache struct {
    mu    sync.RWMutex
    items map[string]string
}

func (c *Cache) Set(key, value string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.items == nil {
        c.items = make(map[string]string)
    }
    c.items[key] = value
}
```


<a id="go-receivers"></a>

### Pointer vs Value Receivers

Use pointer receivers when the method mutates state, the struct is large, or for consistency when other methods on the type use pointer receivers. Use value receivers for small, immutable types like enums or thin wrappers.

A type containing a mutex or another synchronization primitive that must not be copied after use needs pointer semantics even if it is small. Copying a struct also does not deeply copy its slices, maps, or pointers; decide whether the resulting aliasing is part of the API contract.

```go
// Pointer: mutates state or large struct
func (s *Service) Process(ctx context.Context, id string) error { ... }

// Value: small, immutable type
func (s Status) String() string { return string(s) }
```

If any method on a type uses a pointer receiver, all methods on that type should use pointer receivers for consistency.


Sources: [reference 1](https://pkg.go.dev/sync), [reference 2](https://go.dev/wiki/CodeReviewComments#receiver-type).

<a id="go-mutable-input-ownership"></a>

## Ownership of mutable inputs and returned values

For a constructor, option, or public method that accepts a slice, map, pointer, header collection, or buffer, decide whether the callee borrows it for the call, retains it, or takes ownership. State the choice where callers could otherwise mutate shared storage unexpectedly. A copied struct may still share all of those underlying values.

Prefer copying small mutable configuration at the boundary when the instance should remain stable after construction. Copy only as deeply as the promised isolation requires. For large buffers or performance-sensitive paths, an explicit ownership transfer can be clearer than a hidden copy. Do not add indiscriminate deep copies without a contract that needs them.

```go
func WithDefaultHeaders(headers http.Header) Option {
    snapshot := headers.Clone()

    return func(client *Client) {
        client.defaultHeaders = snapshot.Clone()
    }
}
```

This option snapshots at creation and gives each client its own header map if the option is reused. If borrowing is intentional instead, document when the caller may mutate the value again. Apply the same reasoning to returned slices and pointers: callers need to know whether they received a view, a snapshot, or mutable shared state.

Synchronize shared mutable state according to its contract. Returning a pointer under a lock does not keep later caller mutations protected. Read [shared state](concurrency.md#go-shared-state) and [receiver semantics](structs.md#go-receivers) when ownership and concurrent access interact.

State the copy depth and [nil/empty collection contract](types.md#go-collection-contracts) separately; a shallow copy does not isolate objects referenced by its elements.


Sources: [reference 1](https://pkg.go.dev/net/http@go1.26.1#Header.Clone), [reference 2](https://go.dev/wiki/CodeReviewComments#copying).
