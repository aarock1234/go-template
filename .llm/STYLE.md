# Go Style Guide

Project-agnostic Go style guide. Favor idiomatic Go over clever abstractions.

**Minimum Go version**: 1.22+ (examples use range-over-int, per-iteration loop variables, enhanced ServeMux routing, and iterators from 1.23+).
**Recommended Go version**: 1.26.0 (latest stable version)

## Philosophy

Write boring code. Prefer explicit over implicit. Optimize for reading, not writing. Design zero values to work without initialization. Keep functions small and interfaces smaller.

## Quick Reference

### Tools

```bash
go vet ./...                 # static analysis
gofmt -l .                   # check formatting
go mod tidy                  # sync dependencies
go test ./...                # run all tests
go test -race ./...          # with race detection
go build -o bin/app ./cmd/app
```

### Import Order

Four groups, blank line between each: stdlib, external, internal, side-effect.

```go
import (
    "context"
    "errors"
    "fmt"

    "github.com/google/uuid"

    "project/pkg/client"
    "project/pkg/service"

    _ "project/pkg/log"  // side-effect imports: own group with comment
)
```

Side-effect imports (`_`) go in a fourth group so `gofmt` won't sort them into the middle of internal imports.

### Naming Conventions

| Category       | Convention                          | Examples                            |
| -------------- | ----------------------------------- | ----------------------------------- |
| Exported types | PascalCase                          | `Storage`, `UserEvent`, `UserID`    |
| Unexported     | camelCase                           | `processItem`, `defaultTimeout`     |
| Interfaces     | PascalCase; `-er` for single-method | `Reader`, `Writer`, `Processor`     |
| Constants      | PascalCase                          | `DefaultTimeout`, `MaxRetries`      |
| Acronyms       | Consistent case                     | `userID`, `client`, `Client`        |
| Files          | snake_case                          | `item_service.go`, `http_client.go` |
| Packages       | lowercase, single-word              | `rotate`, `auth`, `client`          |

### Package Naming

Short, lowercase, single-word. Name packages after the domain or concept they represent. The package name is part of the API — it provides context to everything it exports, so design the two together.

```go
// GOOD: short, descriptive, domain-oriented
package rotate
package auth
package client

// BAD: generic, multi-word, or utility-dump names
package utils
package helpers
package common
package httpHelpers
```

Avoid stuttering — the package name and its exports are read together. Let the package name carry context so exports don't repeat it.

```go
// GOOD: reads naturally
rotate.File       // not rotate.FileRotator
http.Client       // not http.Client
auth.Token        // not auth.AuthToken

// BAD: package name repeated in export
rotate.FileRotator
auth.AuthToken
http.Client
```

## Error Handling

### Basic Pattern

Return errors as last value. Check immediately. Wrap with context using `fmt.Errorf` and `%w`.

```go
func (s *Service) Get(ctx context.Context, id string) (*Item, error) {
    item, err := s.repo.Get(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get item %s: %w", id, err)
    }

    return item, nil
}
```

Error messages: lowercase, no punctuation, add context. Handle an error or propagate it — never both. Logging and returning creates duplicate noise up the call chain.

```go
// BAD: logs and returns — caller will likely log again
if err != nil {
    s.logger.Error("failed to get item", "error", err)
    return nil, fmt.Errorf("get item: %w", err)
}

// GOOD: propagate with context, let the caller decide
if err != nil {
    return nil, fmt.Errorf("get item %s: %w", id, err)
}
```

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

### Custom Error Types

For errors that carry additional data. Check with `errors.As`.

```go
type ValidationError struct {
    Field string
    Reason string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed: %s: %s", e.Field, e.Reason)
}

// Usage
var valErr *ValidationError
if errors.As(err, &valErr) {
    log.Error("validation failed", "field", valErr.Field)
}
```

### When to Panic

Only for truly unrecoverable situations: `init()` setup failures or programmer errors (impossible states, violated invariants). Never in library code for operational errors.

```go
// init: include the error for debuggability
func init() {
    if err := setupTransport(); err != nil {
        panic(fmt.Sprintf("transport initialization failed: %v", err))
    }
}

// Impossible state after exhaustive handling
switch status {
case StatusActive, StatusPending, StatusCompleted:
    // ...
default:
    panic(fmt.Sprintf("unhandled status: %v", status))
}
```

## Struct Patterns

### Constructor Pattern

```go
type Service struct {
    repo   Repository
    logger *slog.Logger
    timeout time.Duration
}

func NewService(repo Repository, logger *slog.Logger) *Service {
    return &Service{
        repo:    repo,
        logger:  logger,
        timeout: 10 * time.Second,
    }
}
```

### Functional Options Pattern

For complex configuration.

```go
type Option func(*Service)

func WithTimeout(d time.Duration) Option {
    return func(s *Service) { s.timeout = d }
}

func NewService(repo Repository, opts ...Option) *Service {
    s := &Service{
        repo:    repo,
        timeout: 10 * time.Second,
    }
    for _, opt := range opts {
        opt(s)
    }

    return s
}

// Usage
svc := NewService(repo, WithTimeout(5*time.Second))
```

### Struct Embedding

For shared behavior across implementations.

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
    // PDF-specific logic
}

// Note: if PDFProcessor later defines its own Validate method,
// it silently shadows BaseProcessor.Validate.
```

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

## Type Safety & Interfaces

### Never Use `any` Unless Absolutely Necessary

Order of preference: Fully typed → Generics → Semi-typed → `any` (last resort)

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

func Largest[T Sizer](items []T) T {
    max := items[0]
    for _, item := range items[1:] {
        if item.Size() > max.Size() {
            max = item
        }
    }
    return max
}

// GOOD: interface constraints
type HasID interface {
    GetID() string
}

func FindItem[T HasID](items []T, id string) *T {
    for i := range items {
        if items[i].GetID() == id {
            return &items[i]
        }
    }
    return nil
}

// GOOD: multiple constraints
type Identifiable interface {
    GetID() string
}

type Timestamped interface {
    GetCreatedAt() time.Time
}

func SortByCreation[T interface{ Identifiable; Timestamped }](items []T) []T {
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
// common transformations. Use judiciously — a plain for loop
// is often clearer for simple cases.
func Map[T, U any](items []T, fn func(T) U) []U {
    result := make([]U, len(items))
    for i, item := range items {
        result[i] = fn(item)
    }
    return result
}

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

### Order of Preference Examples

```go
// 1. BEST: Fully typed
type ProcessorConfig struct {
    Timeout  time.Duration
    Retries  int
    Endpoint string
}

// 2. ACCEPTABLE: Semi-typed with known key types
type ProcessorRegistry map[string]ProcessorConfig
type HandlerMap map[string]func(context.Context, Item) error

// 3. ACCEPTABLE: Semi-typed with union-like values
type Status string

const (
    StatusPending    Status = "pending"
    StatusProcessing Status = "processing"
    StatusCompleted  Status = "completed"
)

type StatusMap map[string]Status

// 4. LAST RESORT: Semi-typed with any
type DynamicHandlers map[string]any // Only when handlers have varying signatures

// 5. AVOID: Fully untyped
// type BadMap map[any]any // DON'T DO THIS
```

### JSON: Always Use Structs, Not Maps

```go
// BAD: map[string]any
func Handle(data map[string]any) error {
    name, ok := data["name"].(string)  // type assertions everywhere
    if !ok {
        return errors.New("invalid name")
    }
}

// GOOD: struct types
type Request struct {
    Name  string `json:"name"`
    Age   int    `json:"age"`
    Email string `json:"email,omitempty"`
}

func Handle(req Request) error {
    // type-safe, validated at unmarshal time
}
```

Only use `map[string]any` when structure is truly dynamic (plugin configs, user-defined metadata).

### Interface Design

Define interfaces where used (consumer side). Keep them small. Accept interfaces, return concrete types.

```go
// In service package
type Repository interface {
    Get(ctx context.Context, id string) (*Item, error)
    Save(ctx context.Context, item *Item) error
}

type Service struct {
    repo Repository  // accepts interface
}

func NewService(repo Repository) *Service {  // returns concrete type
    return &Service{
        repo: repo,
    }
}
```

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

### Verify Interface Implementation

```go
var _ Repository = (*PostgresRepo)(nil)  // compile-time check
```

## Concurrency Patterns

Prefer synchronization primitives over channels for simple mutual exclusion. Channels are for communication and orchestration; `sync.Mutex` is for protecting shared state. Using the simpler tool makes intent clearer.

### Worker Pool

```go
func (s *Service) ProcessBatch(ctx context.Context, items []Item) error {
    const concurrency = 10
    taskCh := make(chan Item, concurrency*2)
    var wg sync.WaitGroup

    // Start workers
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for item := range taskCh {
                if err := s.process(ctx, item); err != nil {
                    s.logger.Error("process failed", "error", err)
                }
            }
        }()
    }

    // Send work
    go func() {
        defer close(taskCh)
        for _, item := range items {
            select {
            case <-ctx.Done():
                return
            case taskCh <- item:
            }
        }
    }()

    wg.Wait()
    return ctx.Err()
}

// Note: individual item errors are logged, not returned. For operations
// where errors must be collected, prefer errgroup or aggregate into a slice.
```

### Parallel Fetch with Result Channels

```go
func (s *Service) FetchBoth(ctx context.Context, id string) (*Data, error) {
    type result struct {
        data *Response
        err  error
    }

    ch1 := make(chan result, 1)
    ch2 := make(chan result, 1)

    go func() {
        data, err := s.fetchOne(id)
        ch1 <- result{data, err}
    }()

    go func() {
        data, err := s.fetchTwo(id)
        ch2 <- result{data, err}
    }()

    var r1, r2 *Response
    for range 2 {
        select {
        case res := <-ch1:
            if res.err != nil {
                return nil, res.err
            }
            r1 = res.data
        case res := <-ch2:
            if res.err != nil {
                return nil, res.err
            }
            r2 = res.data
        }
    }

    return &Data{
        One: *r1,
        Two: *r2,
    }, nil
}

// For simple parallel fetches like this, errgroup (below) is often cleaner.
```

### Errgroup for Concurrent Operations

```go
import "golang.org/x/sync/errgroup"

func (s *Service) ProcessAll(ctx context.Context, items []Item) error {
    g, ctx := errgroup.WithContext(ctx)

    for _, item := range items {
        g.Go(func() error {
            return s.process(ctx, item)
        })
    }

    return g.Wait()
}
```

### Thread-Safe State

```go
type Cycle[T any] struct {
    items []T
    index int
    mu    sync.Mutex
}

func (c *Cycle[T]) Next() T {
    c.mu.Lock()
    defer c.mu.Unlock()

    item := c.items[c.index]
    c.index = (c.index + 1) % len(c.items)
    return item
}
```

## HTTP Client Pattern

Structure: `pkg/client/{client.go, proxies.go, cookies.go}`

**Note:** This section uses `github.com/enetx/http` (a fork of `net/http`). Adapt imports for your project.

Wrap `*http.Client` for cleaner API. Include cookie jar. Always wrap errors with context.

### client.go

```go
package client

import (
    "fmt"
    "github.com/enetx/http"
    "net/http/cookiejar"
    "net/url"
)

type Client struct {
    inner *http.Client
    proxy *url.URL
}

func NewClient(proxy *url.URL) (*Client, error) {
    client, err := newClient(proxy)
    if err != nil {
        return nil, fmt.Errorf("creating http client: %w", err)
    }

    return &Client{
        inner: client,
        proxy: proxy,
    }, nil
}

func newClient(proxy *url.URL) (*http.Client, error) {
    jar, err := cookiejar.New(nil)
    if err != nil {
        return nil, fmt.Errorf("creating cookie jar: %w", err)
    }

    return &http.Client{
        Transport: &http.Transport{
            Proxy: http.ProxyURL(proxy),
        },
        Jar: jar,
    }, nil
}
```

### proxies.go

```go
package client

import (
    "bufio"
    "fmt"
    "net/url"
    "os"
    "strings"
)

func ParseProxy(line string) (*url.URL, error) {
    parts := strings.Split(line, ":")
    if len(parts) != 2 && len(parts) != 4 {
        return nil, fmt.Errorf("invalid proxy: %s", line)
    }

    proxy := &url.URL{
        Scheme: "http",
        Host:   parts[0] + ":" + parts[1],
    }

    if len(parts) == 4 {
        proxy.User = url.UserPassword(parts[2], parts[3])
    }

    return proxy, nil
}

func ImportProxies(filename string) ([]*url.URL, error) {
    f, err := os.Open(filename)
    if err != nil {
        return nil, fmt.Errorf("opening proxy file: %w", err)
    }
    defer f.Close()

    var proxies []*url.URL
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        proxy, err := ParseProxy(scanner.Text())
        if err != nil {
            return nil, fmt.Errorf("parsing proxy line %q: %w", scanner.Text(), err)
        }
        proxies = append(proxies, proxy)
    }

    if err := scanner.Err(); err != nil {
        return nil, fmt.Errorf("scanning proxy file: %w", err)
    }

    return proxies, nil
}
```

### cookies.go

```go
package client

import (
    "github.com/enetx/http"
    "net/url"
)

func (c *Client) SetCookies(u *url.URL, cookies []*http.Cookie) {
    c.inner.Jar.SetCookies(u, cookies)
}
```

### Usage Example

```go
type APIClient struct {
    http    *client.Client
    baseURL string
}

func NewAPIClient(baseURL string, proxy *url.URL) (*APIClient, error) {
    client, err := client.NewClient(proxy)
    if err != nil {
        return nil, fmt.Errorf("creating http client: %w", err)
    }

    return &APIClient{
        http:    client,
        baseURL: baseURL,
    }, nil
}

func (c *APIClient) Get(ctx context.Context, path string, response any) error {
    req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
    if err != nil {
        return fmt.Errorf("creating request: %w", err)
    }

    req.Header.Set("Accept", "application/json")

    res, err := c.http.inner.Do(req)
    if err != nil {
        return fmt.Errorf("executing request: %w", err)
    }
    defer res.Body.Close()

    if res.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status: %d", res.StatusCode)
    }

    if err := json.NewDecoder(res.Body).Decode(response); err != nil {
        return fmt.Errorf("decoding response: %w", err)
    }

    return nil
}
```

## Code Organization

### Package Structure

```
project/
├── cmd/
│   └── server/
│       └── main.go        # entrypoint
└── pkg/
    ├── handler/            # HTTP handlers
    ├── service/            # business logic
    ├── repository/         # data access
    ├── model/              # shared types
    └── log/                # side-effect init
```

`pkg/` is the default for all project packages. Only use `internal/` when the module is published for external consumers — this will be explicitly communicated.

### Layered Architecture

**Repository Layer** - Data access

```go
type Repository interface {
    Get(ctx context.Context, id string) (*Item, error)
    Save(ctx context.Context, item *Item) error
}

type PgRepo struct {
    db     *sql.DB
    logger *slog.Logger
}

func NewRepository(db *sql.DB, logger *slog.Logger) *PgRepo {
    return &PgRepo{
        db:     db,
        logger: logger,
    }
}
```

**Service Layer** - Business logic

```go
type Service struct {
    repo   Repository
    logger *slog.Logger
}

func NewService(repo Repository, logger *slog.Logger) *Service {
    return &Service{
        repo:   repo,
        logger: logger,
    }
}

func (s *Service) Process(ctx context.Context, id string) error {
    item, err := s.repo.Get(ctx, id)
    if err != nil {
        return fmt.Errorf("get item: %w", err)
    }

    // business logic here

    return s.repo.Save(ctx, item)
}
```

**Handler Layer** - HTTP/transport

```go
type Handler struct {
    service *Service
    logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
    return &Handler{
        service: service,
        logger:  logger,
    }
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")

    item, err := h.service.Get(r.Context(), id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            http.Error(w, "not found", http.StatusNotFound)
            return
        }
        h.logger.Error("get failed", "error", err)
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(item); err != nil {
        h.logger.Error("encode response", "error", err)
    }
}
```

### Dependency Wiring

Prefer manual wiring. Keep it explicit in `main.go`.

```go
func main() {
    logger := slog.Default()

    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        logger.Error("failed to open database", "error", err)
        os.Exit(1)
    }
    defer db.Close()

    repo := repository.New(db, logger)
    service := service.New(repo, logger)
    handler := handler.New(service, logger)

    mux := http.NewServeMux()
    mux.HandleFunc("GET /items/{id}", handler.Get)
    mux.HandleFunc("POST /items", handler.Create)

    server := &http.Server{
        Addr:         ":8080",
        Handler:      mux,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
    }

    logger.Info("server starting", "port", "8080")
    if err := server.ListenAndServe(); err != nil {
        logger.Error("server error", "error", err)
    }
}
```

## Context Propagation

First parameter for functions doing I/O. Pass through the entire call chain. Use for cancellation, timeouts, and request-scoped values.

**Never store `context.Context` in a struct.** Pass it as a function argument so each call gets its own cancellation scope.

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

Use `context.Background()` at the top of call chains (`main()`, test setup). Use `context.TODO()` as a placeholder when unsure which context to use. Never pass `nil` — use `context.TODO()` instead.

## Logging

Use `log/slog` with structured logging. Initialize default logger via side-effect import.

```go
// pkg/log/log.go
package log

import (
    "log/slog"
    "os"

    "github.com/lmittmann/tint"
)

func init() {
    slog.SetDefault(slog.New(tint.NewHandler(os.Stdout, nil)))
}
```

```go
// main.go
import (
    "log/slog"

    _ "project/pkg/log"
)

func main() {
    slog.Info("starting application")
}
```

Use structured attributes. All log messages lowercase.

```go
s.logger.InfoContext(ctx, "item processed",
    slog.Duration("duration", elapsed),
    slog.String("item_id", id),
)

s.logger.ErrorContext(ctx, "failed to save",
    slog.Any("error", err),
)
```

## Constants and Enums

Use typed constants for enum behavior. Use `iota` for sequential integers.

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

## Iterators

Use `iter.Seq` and `iter.Seq2` to make custom types rangeable. Prefer iterators over returning full slices when the caller may not need all elements.

```go
import "iter"

// Seq for single-value iteration
func (s *Set[E]) All() iter.Seq[E] {
    return func(yield func(E) bool) {
        for e := range s.m {
            if !yield(e) {
                return
            }
        }
    }
}

// Seq2 for key-value iteration
func (s *Store) All() iter.Seq2[string, Item] {
    return func(yield func(string, Item) bool) {
        for k, v := range s.items {
            if !yield(k, v) {
                return
            }
        }
    }
}

// Usage: range directly over custom types
for item := range mySet.All() {
    process(item)
}

for key, item := range store.All() {
    fmt.Println(key, item)
}
```

Compose iterators for lazy transformation chains instead of allocating intermediate slices.

```go
// Lazy filtering — no intermediate allocation
func FilterIter[V any](seq iter.Seq[V], pred func(V) bool) iter.Seq[V] {
    return func(yield func(V) bool) {
        for v := range seq {
            if pred(v) {
                if !yield(v) {
                    return
                }
            }
        }
    }
}

// Usage
for item := range FilterIter(mySet.All(), isActive) {
    process(item)
}
```

## Testing

See [TESTING.md](TESTING.md) for comprehensive testing patterns.

Quick commands:

```bash
go test ./...                     # all tests
go test -v ./pkg/storage/...      # verbose, specific package
go test -run TestName ./...       # specific test
go test -race ./...               # race detection
go test -cover ./...              # coverage
```

## Development Workflow

### Dependency Management

Prefer `go mod tidy` over `go get` for adding dependencies.

**Workflow:**

1. Add import to your `.go` file: `import "github.com/user/package"`
2. Run `go mod tidy` to fetch and add to `go.mod`

```bash
# PREFERRED: Import in code, then tidy
go mod tidy

# Use go get for specific versions
go get github.com/user/package@v1.2.3

# Use go get for upgrading dependencies
go get -u github.com/user/package

# Update all dependencies to latest minor/patch
go get -u ./...
```

**Why prefer `go mod tidy`:**

- Automatically manages both additions and removals
- Ensures `go.mod` and `go.sum` are in sync
- Removes unused dependencies
- Cleaner workflow: write code first, manage deps second
- Less error-prone than manual `go get` for each package

### Static Analysis

Use `go vet` to check for suspicious code without building. Faster than full build for catching common errors.

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
- In CI/CD before running tests

## Additional Conventions

### Doc Comments

Exported types and functions must have doc comments. Comments begin with the name of the thing being described and are complete sentences ending with a period.

```go
// Service processes items according to business rules.
type Service struct { ... }

// Process validates and persists the given item. It returns
// an error if the item fails validation or cannot be saved.
func (s *Service) Process(ctx context.Context, item *Item) error { ... }
```

### Receiver Naming

Use one or two letter abbreviations consistent across all methods of a type. Do not use `this` or `self`.

```go
// GOOD: short, consistent
func (s *Service) Get(ctx context.Context, id string) (*Item, error) { ... }
func (s *Service) Save(ctx context.Context, item *Item) error { ... }

// BAD: verbose, Java-style
func (service *Service) Get(...) { ... }
func (self *Service) Save(...) { ... }
```

### Graceful Shutdown

Production servers should handle OS signals for clean shutdown.

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    server := &http.Server{
        Addr:    ":8080",
        Handler: mux,
    }

    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            slog.ErrorContext(ctx, "server error", "error", err)
        }
    }()

    <-ctx.Done()
    slog.InfoContext(ctx, "shutting down")

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := server.Shutdown(shutdownCtx); err != nil {
        slog.ErrorContext(ctx, "shutdown error", "error", err)
    }
}
```
