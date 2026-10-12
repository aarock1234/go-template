// Package log initializes the default slog logger with tint for colorized,
// structured output and injects request-scoped values from context.
//
// Importing the package installs the logger. Its level starts at the
// LOG_LEVEL process environment variable (default info) and can be changed
// with SetLevel once configuration is loaded.
package log

import (
	"context"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

type contextKey int

const (
	idempotencyKey contextKey = iota
	requestIDKey
)

// level is the minimum level of the default logger.
var level slog.LevelVar

func init() {
	// An unset or invalid LOG_LEVEL keeps the info level; configuration
	// loading validates the value later.
	_ = level.UnmarshalText([]byte(os.Getenv("LOG_LEVEL")))

	w := os.Stdout

	slog.SetDefault(slog.New(&ContextHandler{
		Handler: tint.NewTextHandler(colorable.NewColorable(w), &tint.Options{
			TimeFormat: "Mon, Jan 2 2006, 3:04:05 pm MST",
			NoColor:    !isatty.IsTerminal(w.Fd()),
			Level:      &level,
		}),
	}))
}

// SetLevel changes the minimum level of the default logger.
func SetLevel(l slog.Level) {
	level.Set(l)
}

// WithIdempotencyKey returns a copy of ctx carrying the given idempotency key.
func WithIdempotencyKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, idempotencyKey, key)
}

// WithRequestID returns a copy of ctx carrying the given request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext returns the request ID carried by ctx, or an empty
// string if none is set.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)

	return id
}

// ContextHandler wraps an slog.Handler to add request-scoped values from
// the context to every record.
type ContextHandler struct {
	slog.Handler
}

// Handle adds the context's request ID and idempotency key to r, then
// passes it to the wrapped handler.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if key, ok := ctx.Value(idempotencyKey).(string); ok {
		r.AddAttrs(slog.String("idempotency_key", key))
	}

	if id, ok := ctx.Value(requestIDKey).(string); ok {
		r.AddAttrs(slog.String("request_id", id))
	}

	return h.Handler.Handle(ctx, r)
}

// WithAttrs returns a ContextHandler whose wrapped handler includes attrs,
// so loggers derived with slog.With keep adding context values.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{
		Handler: h.Handler.WithAttrs(attrs),
	}
}

// WithGroup returns a ContextHandler whose wrapped handler nests later
// attributes under name.
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return &ContextHandler{
		Handler: h.Handler.WithGroup(name),
	}
}
