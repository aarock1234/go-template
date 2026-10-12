# Consumer interfaces and composition

[Back to the Go entrypoint](../SKILL.md). Read the sections matching the current change; follow crosslinks when the change crosses a boundary.

- [Interface Design](#go-consumer-interfaces)
- [Interface Composition](#go-interface-composition)
- [Verify Interface Implementation](#go-interface-assertions)

<a id="go-consumer-interfaces"></a>

### Interface Design

Define a small interface at the consumer when an actual use establishes the required behavior. Prefer concrete return types so implementation APIs can evolve without broad interface changes. These are design defaults: an established abstraction may legitimately return an interface, and a single concrete dependency need not gain an interface before it has a useful contract.

```go
// In service package
type Repository interface {
    Get(ctx context.Context, id string) (*Item, error)
    Save(ctx context.Context, item *Item) error
}

type Service struct {
    repo Repository  // accepts interface
}

func New(repo Repository) *Service {  // returns concrete type
    return &Service{
        repo: repo,
    }
}
```

An interface may hold a typed nil pointer and still compare unequal to nil. If a dependency excludes nil implementations, document that contract or validate a concrete pointer before interface conversion. An interface-level `dependency == nil` guard cannot establish that every implementation is usable; avoid generic reflection validation without a real dynamic contract that needs it.

Source: [Go nil interfaces](https://go.dev/doc/faq#nil_error).


Sources: [reference 1](https://go.dev/wiki/CodeReviewComments#interfaces).

<a id="go-interface-composition"></a>

### Interface Composition

```go
type Reader interface {
    Read(ctx context.Context, id string) ([]byte, error)
}

type Writer interface {
    Write(ctx context.Context, id string, data []byte) error
}

// Compose interfaces
type ReadWriter interface {
    Reader
    Writer
}

type Storage struct {
    rw ReadWriter  // accepts composed interface
}
```


<a id="go-interface-assertions"></a>

### Verify Interface Implementation

```go
var _ Repository = (*PostgresRepo)(nil)  // compile-time check
```

