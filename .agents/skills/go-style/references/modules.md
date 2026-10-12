# Imports, names, spacing, and comments

[Back to the Go entrypoint](../SKILL.md). Read the sections matching the current change; follow crosslinks when the change crosses a boundary.

- [Import Order](#go-import-order)
- [Naming Conventions](#go-naming)
- [Package Naming](#go-package-names)
- [Newline Spacing](#go-spacing)
- [Doc Comments](#go-doc-comments)
- [Receiver Naming](#go-receiver-names)

<a id="go-import-order"></a>

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


<a id="go-naming"></a>

### Naming Conventions

| Category       | Convention                          | Examples                            |
| -------------- | ----------------------------------- | ----------------------------------- |
| Exported types | PascalCase                          | `Storage`, `UserEvent`, `UserID`    |
| Unexported     | camelCase                           | `processItem`, `defaultTimeout`     |
| Interfaces     | PascalCase; `-er` for single-method | `Reader`, `Writer`, `Processor`     |
| Constants      | Exported PascalCase; unexported camelCase | `DefaultTimeout`, `defaultTimeout` |
| Initialisms    | Consistent case; lowercase at an unexported start | `httpClient`, `userID`, `HTTPClient`, `parseURL` |
| Files          | snake_case                          | `item_service.go`, `http_client.go` |
| Packages       | lowercase, single-word              | `rotate`, `auth`, `client`          |

The original “all-caps both cases” shorthand must not accidentally export an identifier. Use `httpClient` for an unexported name beginning with HTTP; keep an interior initialism intact in `userID`. Apply these conventions to hand-written identifiers while preserving generated naming contracts.


Sources: [reference 1](https://go.dev/wiki/CodeReviewComments#initialisms).

<a id="go-package-names"></a>

### Package Naming

Short, lowercase, single-word. Name packages after the domain or concept they represent. The package name is part of the API; it provides context to everything it exports, so design the two together.

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

Avoid stuttering; the package name and its exports are read together. Let the package name carry context so exports don't repeat it.

```go
// GOOD: reads naturally
rotate.File       // not rotate.FileRotator
http.Client       // not http.HTTPClient
auth.Token        // not auth.AuthToken

// BAD: package name repeated in export
rotate.FileRotator
auth.AuthToken
http.HTTPClient
```


<a id="go-additional-conventions"></a>

## Additional Conventions

Use the [naming and import conventions](modules.md), then [receiver guidance](structs.md#go-receivers) for method changes.


<a id="go-spacing"></a>

### Newline Spacing

Use blank lines to separate logical groups within a function body. Separate a return from preceding work in the same block; a return that is the first or only statement of a guard clause does not need an empty line after the opening brace. A compact one-statement method or option closure can stay compact.

Keep closely related statements together: a variable definition and the loop or condition that immediately uses it, or a fetch and the assignment from its result. When several unrelated statements stack up, separate them by intent rather than enforcing an arbitrary statement count. This resolves the earlier “every return” wording in favor of the guide's own guard-clause examples.

```go
// GOOD: grouped by intent, return has breathing room
func Process(ctx context.Context, items []Item) (int, error) {
    total := 0
    for _, item := range items {
        total += item.Value
    }

    if err := validate(total); err != nil {
        return 0, fmt.Errorf("validating total: %w", err)
    }

    return total, nil
}

// BAD: everything jammed together
func Process(ctx context.Context, items []Item) (int, error) {
    total := 0
    for _, item := range items {
        total += item.Value
    }
    if err := validate(total); err != nil {
        return 0, fmt.Errorf("validating total: %w", err)
    }
    return total, nil
}
```


<a id="go-doc-comments"></a>

### Doc Comments

Document exported identifiers, including types, functions, methods, constants, variables, fields, and interface methods. Comments begin with the name of the thing being described and are complete sentences ending with a period.

```go
// Service processes items according to business rules.
type Service struct { ... }

// Process validates and persists the given item. It returns
// an error if the item fails validation or cannot be saved.
func (s *Service) Process(ctx context.Context, item *Item) error { ... }
```

A declaration group may share a comment explaining its common contract; add individual comments where a member has distinct meaning, units, lifecycle, or constraints. Describe behavior and caller obligations rather than repeating an obvious name. This preserves the original summary's scope beyond only types and functions.


<a id="go-receiver-names"></a>

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

