// Package log initializes the default slog logger with tint for colorized,
// structured output and injects request-scoped values from context.
package log

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

// ContextHandler wraps an slog.Handler to inject request-scoped values
// from the context into every log record.
type ContextHandler struct {
	slog.Handler
}

// Handle logs a slog.Record with request-scoped values from the context.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if key, ok := ctx.Value("idempotency_key").(string); ok {
		r.AddAttrs(slog.String("idempotency_key", key))
	}

	return h.Handler.Handle(ctx, r)
}

func init() {
	w := os.Stdout

	slog.SetDefault(slog.New(
		&ContextHandler{
			Handler: tint.NewHandler(colorable.NewColorable(w), &tint.Options{
				TimeFormat: "Mon, Jan 2 2006, 3:04:05 pm MST",
				NoColor:    !isatty.IsTerminal(w.Fd()),
				Level:      logLevelFromEnv(),
			}),
		}),
	)
}

func logLevelFromEnv() slog.Level {
	switch strings.ToLower(
		strings.TrimSpace(
			os.Getenv("LOG_LEVEL"),
		),
	) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}
