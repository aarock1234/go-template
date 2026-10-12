# Package layout and dependency wiring

[Back to the Go entrypoint](../SKILL.md). Read the sections matching the current change; follow crosslinks when the change crosses a boundary.

- [Package Structure](#go-package-structure)
- [Layered Architecture](#go-layers)
- [Dependency Wiring](#go-dependency-wiring)

<a id="go-organization"></a>

## Code Organization

Use [package boundaries](#go-package-structure) and [explicit dependency wiring](#go-dependency-wiring); add layers only for an existing ownership or change boundary.


<a id="go-package-structure"></a>

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

The template uses `pkg/` as its default project layout; preserve that convention when working in a repository built from it. This is a layout preference, not a Go restriction. `internal/` enforces an import boundary and is also valid for private commands and servers. Use the repository's established boundary policy, and choose `internal/` when packages are deliberately unavailable to external importers. Do not relocate an existing package tree merely to satisfy a generic layout slogan.


Sources: [reference 1](https://go.dev/doc/modules/layout).

<a id="go-layers"></a>

### Layered Architecture

Use these layers when transport, business rules, and storage have distinct responsibilities. Keep a small command direct if separate layers would only forward calls. The repository contract belongs to the consuming service; the examples show the contract beside its implementation to explain the dependency, not to require an implementer-owned interface.

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

func New(db *sql.DB, logger *slog.Logger) *PgRepo {
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

func New(repo Repository, logger *slog.Logger) *Service {
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

func New(service *Service, logger *slog.Logger) *Handler {
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
        h.logger.ErrorContext(r.Context(), "get failed", slog.Any("error", err))
        http.Error(w, "internal error", http.StatusInternalServerError)

        return
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(item); err != nil {
        h.logger.Error("encode response", "error", err)
    }
}
```

Query-layer boundaries also own [transactions and cursor completion](data.md); keep those contracts visible when adding a repository.


Sources: [reference 1](https://go.dev/wiki/CodeReviewComments#interfaces).

<a id="go-dependency-wiring"></a>

### Dependency Wiring

Prefer manual wiring. Keep it explicit in `main.go`. Extract the body into a `run()` function that returns an error so that `defer` statements execute on all exit paths. Reserve `os.Exit` for the top-level `main()`.

```go
func main() {
    if err := run(); err != nil {
        slog.Error("fatal error", "error", err)
        os.Exit(1)
    }
}

func run() error {
    logger := slog.Default()

    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        return fmt.Errorf("open database: %w", err)
    }
    defer func() { _ = db.Close() }()

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
    if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        return fmt.Errorf("server: %w", err)
    }

    return nil
}
```

This example demonstrates explicit dependency construction. Use [graceful shutdown](lifecycle.md#go-shutdown) for a long-lived server's signal handling, listener ownership, and bounded cleanup. `sql.Open` prepares a database handle; call `PingContext` when startup must establish that the configured database is reachable. Do not treat `http.ErrServerClosed` after an intentional shutdown as an unexpected listener failure.


Sources: [reference 1](https://pkg.go.dev/net/http@go1.26.1#Server.ListenAndServe), [reference 2](https://pkg.go.dev/database/sql#Open).
