# Types, JSON, and constants

[Back to the Go entrypoint](../SKILL.md). Read the sections matching the current change; follow crosslinks when the change crosses a boundary.

- [Typed values and intentional dynamic boundaries](#go-typed-values)
- [Order of Preference Examples](#go-type-preferences)
- [Known JSON shapes and domain validation](#go-json-boundaries)
- [Optional Pointer Fields](#go-optional-fields)
- [Constants and Enums](#go-constants)
- [Nil and empty collection contracts](#go-collection-contracts)

<a id="go-type-safety"></a>

## Type Safety & Interfaces

Start with [typed boundaries](#go-typed-values), then [JSON validation](#go-json-boundaries), [absence](#go-optional-fields), or [collection shape](#go-collection-contracts) as needed.


<a id="go-typed-values"></a>

### Typed values and intentional dynamic boundaries

For domain values, prefer a concrete type; use generics for a useful algorithm or container shared across types; use typed maps for genuinely keyed collections. Reserve `any` for boundaries whose contract is intentionally dynamic, such as a serialization adapter. This is a decision order, not a reason to replace a simple interface or a typed map with generics.

If an operation only invokes an established method contract, an interface usually expresses it more directly than a new type parameter. Avoid adding constraints that the implementation does not need. `any` as a generic constraint can preserve the caller's concrete type and is different from discarding that type into an `any` value.

```go
// BAD: any everywhere
func ProcessData(data any) any {
    // No type safety, requires type assertions everywhere
    return data
}

func FindItem(items []any, id string) any {
    for _, item := range items {
        if m, ok := item.(map[string]any); ok {
            if m["id"] == id {
                return item
            }
        }
    }
    return nil
}

// GOOD: type constraints enable meaningful operations
type Sizer interface {
    Size() int
}

func Largest[T Sizer](items []T) (T, bool) {
    var zero T
    if len(items) == 0 {
        return zero, false
    }

    max := items[0]
    for _, item := range items[1:] {
        if item.Size() > max.Size() {
            max = item
        }
    }

    return max, true
}

// GOOD: interface constraints
type HasID interface {
    GetID() string
}

// FindItem returns the first matching element without changing its type.
func FindItem[T HasID](items []T, id string) (T, bool) {
    for _, item := range items {
        if item.GetID() == id {
            return item, true
        }
    }

    var zero T

    return zero, false
}

// GOOD: multiple constraints
type Identifiable interface {
    GetID() string
}

type Timestamped interface {
    GetCreatedAt() time.Time
}

type CreatedEntity interface {
    Identifiable
    Timestamped
}

// Combine constraints when the operation needs both contracts.
// This sorter only reads creation time, so Timestamped is sufficient.
func SortByCreation[T Timestamped](items []T) []T {
    sorted := make([]T, len(items))
    copy(sorted, items)
    sort.Slice(sorted, func(i, j int) bool {
        return sorted[i].GetCreatedAt().Before(sorted[j].GetCreatedAt())
    })

    return sorted
}

// GOOD: generics for container types (e.g., thread-safe wrappers, caches)
type Pair[K, V any] struct {
    Key   K
    Value V
}

// ACCEPTABLE: generic utility functions reduce boilerplate for
// common transformations. Use judiciously; a plain for loop
// is often clearer for simple cases.
// Map allocates a result even for zero elements.
func Map[T, U any](items []T, fn func(T) U) []U {
    result := make([]U, len(items))
    for i, item := range items {
        result[i] = fn(item)
    }

    return result
}

// Filter returns nil when no elements match.
func Filter[T any](items []T, predicate func(T) bool) []T {
    var result []T
    for _, item := range items {
        if predicate(item) {
            result = append(result, item)
        }
    }

    return result
}

// Usage: type inference works automatically
ids := Map(items, func(item Item) string { return item.ID })
activeItems := Filter(items, func(item Item) bool { return item.Active })
```

With `[]*Item`, the lookup result is `*Item`; value elements produce value copies. This is shallow: pointers still refer to the original object. Return an index or `*T` when callers intentionally need input-slice slot access, and state that aliasing contract. Input elements must support the method call, including a defined nil-receiver policy if nil pointer elements are permitted.


Sources: [reference 1](https://go.dev/blog/when-generics).

<a id="go-type-preferences"></a>

### Order of Preference Examples

```go
// 1. Fixed fields
type ProcessorConfig struct {
    Timeout  time.Duration
    Retries  int
    Endpoint string
}

// 2. Keyed values with concrete element types
type ProcessorRegistry map[string]ProcessorConfig
type HandlerMap map[string]func(context.Context, Item) error

// 3. Domain values with explicit validation
type Status string

const (
    StatusPending    Status = "pending"
    StatusProcessing Status = "processing"
    StatusCompleted  Status = "completed"
)

type StatusMap map[string]Status

// 4. Intentionally dynamic values
type DynamicHandlers map[string]any // Only when handlers have varying signatures

// 5. AVOID: Fully untyped
// type BadMap map[any]any // DON'T DO THIS
```

Structs and typed maps describe different data shapes; choose by the contract rather than treating keyed collections as less type-safe. A named enum still needs boundary validation as described in [constants](#go-constants).


<a id="go-json-boundaries"></a>

### Known JSON shapes and domain validation

Use structs for known JSON shapes. Decoding checks representable field types; it does not establish required fields, enum membership, or domain invariants. Validate those explicitly after decoding. Choose pointer or other optional fields when absence must differ from a zero value.

Unknown object keys are ignored by default. Use `Decoder.DisallowUnknownFields` when the boundary intentionally rejects schema extensions. If a boundary accepts exactly one document, reject trailing values; set a size limit for untrusted bodies before decoding. Encoding omission tags do not validate input, and `omitzero` requires Go 1.24+.

```go
// BAD: map[string]any
func Handle(data map[string]any) error {
    name, ok := data["name"].(string)  // type assertions everywhere
    if !ok {
        return errors.New("invalid name")
    }

    _ = name // further work still relies on runtime assertions

    return nil
}

// GOOD: struct types
type Request struct {
    Name      string    `json:"name"`
    Age       int       `json:"age,omitzero"`
    Email     string    `json:"email,omitempty"`
    CreatedAt time.Time `json:"created_at,omitzero"`
}

func Handle(req Request) error {
    if strings.TrimSpace(req.Name) == "" {
        return errors.New("name is required")
    }

    if req.Age < 0 {
        return errors.New("age must not be negative")
    }

    return nil
}
```

Only use `map[string]any` when structure is truly dynamic (plugin configs, user-defined metadata).

On a fresh decode, a pointer field treats an omitted key and explicit JSON `null` as nil. It distinguishes those cases together from a present zero value. PATCH or merge APIs that need separate absent, null, and value states require explicit presence representation or boundary-specific unmarshaling. Decode independent requests into fresh values unless retaining omitted fields is intentional.


Sources: [reference 1](https://pkg.go.dev/encoding/json@go1.26.1#Unmarshal), [reference 2](https://pkg.go.dev/encoding/json@go1.26.1#Decoder), [reference 3](https://go.dev/doc/go1.24#encodingjson).

<a id="go-optional-fields"></a>

### Optional Pointer Fields

Go 1.26 extended `new` to accept an expression, eliminating the need to declare a variable just to take its address. Prefer it for optional pointer fields when the module supports Go 1.26+. On earlier baselines, an addressable local remains appropriate; it is also useful when naming the value explains the calculation.

```go
type Person struct {
    Name string `json:"name"`
    Age  *int   `json:"age,omitempty"`
}

// GOOD: inline with new(expr)
p := Person{
    Name: name,
    Age:  new(yearsSince(born)),
}

// COMPATIBILITY: appropriate before Go 1.26, or when the name adds meaning
age := yearsSince(born)
p := Person{
    Name: name,
    Age:  &age,
}
```

For decoded JSON, a pointer alone does not distinguish a missing key from explicit null; see [presence semantics](#go-json-boundaries).


Sources: [reference 1](https://go.dev/doc/go1.26#language).

<a id="go-constants"></a>

## Constants and Enums

Use typed constants for enum behavior. Prefer string-based enums when values are serialized, logged, or displayed; they're self-documenting. Use integer enums with `iota` when values are purely internal or performance-critical.

A named string or integer type does not constrain values to its declared constants. Validate external values at the boundary and decide whether the zero value is meaningful, unknown, or invalid. A parsing function should return an error for unsupported values rather than relying on a type conversion to validate them.

```go
// String-based enum
type Status string

const (
    StatusPending   Status = "pending"
    StatusActive    Status = "active"
    StatusCompleted Status = "completed"
)

// Integer-based enum
type Priority int

const (
    PriorityLow Priority = iota
    PriorityMedium
    PriorityHigh
    PriorityCritical
)
```


Sources: [reference 1](https://go.dev/ref/spec#Type_definitions).

<a id="go-collection-contracts"></a>

## Nil and empty collection contracts

Choose whether an empty result is nil or allocated when the distinction is observable. Preserve existing API and serialization contracts instead of changing allocation solely for style. A nil map supports reads and iteration but needs allocation before writes.

With `encoding/json`, a nil ordinary slice encodes as `null`, while an allocated empty slice encodes as `[]`; `[]byte` has separate encoding behavior. Normalize at the response boundary when the API promises an array. Omission tags can also make presence part of the contract.

A copy operation should state whether it preserves nilness. `slices.Clone` preserves nilness and copies elements shallowly; pointer, map, and slice elements may still share storage. Read [mutable ownership](structs.md#go-mutable-input-ownership) for the isolation guarantee.


Sources: [JSON encoding](https://pkg.go.dev/encoding/json@go1.26.1#Marshal), [slice cloning](https://pkg.go.dev/slices@go1.26.1#Clone).
