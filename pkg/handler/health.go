package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/aarock1234/go-template/pkg/respond"
)

// Pinger tests connectivity to a backing service.
type Pinger interface {
	Ping(ctx context.Context) error
}

type healthResponse struct {
	Status string `json:"status"`
}

// Live reports that the process is running. It checks no dependencies, so
// an unavailable database never causes a liveness restart.
func Live(w http.ResponseWriter, _ *http.Request) {
	respond.JSON(w, http.StatusOK, healthResponse{
		Status: "ok",
	})
}

// Ready returns a handler that reports whether the service can take
// traffic by pinging its backing service.
func Ready(p Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := p.Ping(r.Context()); err != nil {
			slog.WarnContext(r.Context(), "readiness check failed", slog.Any("error", err))
			respond.Error(w, http.StatusServiceUnavailable, "unavailable")

			return
		}

		respond.JSON(w, http.StatusOK, healthResponse{
			Status: "ok",
		})
	}
}
