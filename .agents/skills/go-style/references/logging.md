# Structured logging

[Back to the Go entrypoint](../SKILL.md). Read the sections matching the current change; follow crosslinks when the change crosses a boundary.

- [Logging](#go-logging)

<a id="go-logging"></a>

## Logging

Use `log/slog` for structured logging. The template installs its process default logger from the `log` package's `init` when the entrypoint imports it, then applies the configured level with `log.SetLevel` after loading configuration. This is an application convention: reusable libraries must not replace the process-wide logger as an import side effect. Explicit initialization in `run` is appropriate when logger configuration can fail or depends on parsed configuration.

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

Context-aware logging supplies a context to the handler; it does not automatically turn context values into attributes. Add useful request identifiers deliberately and redact tokens, passwords, cookies, and credential-bearing URLs before formatting. `slog.LogValuer` can define a safe representation of a sensitive type.

```go
s.logger.InfoContext(ctx, "item processed",
    slog.Duration("duration", elapsed),
    slog.String("item_id", id),
)

s.logger.ErrorContext(ctx, "failed to save",
    slog.Any("error", err),
)
```

A custom `slog.Handler` wrapper must preserve its behavior when `WithAttrs` or `WithGroup` derives a handler. Re-wrap the derived underlying handler instead of relying on promoted methods. Handler methods may run concurrently: synchronize shared mutable configuration and avoid modifying an existing handler when deriving another.

For the template's `ContextHandler`, which embeds `slog.Handler` and augments `Handle`, implement the companion methods:

```go
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{
		Handler: h.Handler.WithAttrs(attrs),
	}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return &ContextHandler{
		Handler: h.Handler.WithGroup(name),
	}
}
```

Preserve any additional wrapper configuration when constructing the derivative. Record cloning is needed when retaining or independently modifying shared copies; it is not a blanket requirement for a single delegated call.

Sources: [slog Handler contract](https://pkg.go.dev/log/slog@go1.26.1#Handler), [handler implementation guidance](https://go.dev/s/slog-handler-guide).


Sources: [reference 1](https://pkg.go.dev/log/slog@go1.26.1#hdr-Contexts), [reference 2](https://pkg.go.dev/log/slog@go1.26.1#LogValuer).
