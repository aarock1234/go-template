// Package handler provides HTTP handlers for the application.
package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/aarock1234/go-template/pkg/apperr"
	"github.com/aarock1234/go-template/pkg/respond"
	"github.com/aarock1234/go-template/pkg/service"
)

// Handler serves HTTP requests using the service layer.
type Handler struct {
	svc *service.Service
}

// New creates a Handler backed by the given service.
func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// GetExample handles GET /api/example and returns the result of the
// example query.
func (h *Handler) GetExample(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Example(r.Context())
	if err != nil {
		handleError(r.Context(), w, err)
		return
	}

	respond.JSON(w, http.StatusOK, exampleResponse{Result: result})
}

type exampleResponse struct {
	Result int32 `json:"result"`
}

// handleError maps application errors to HTTP responses.
func handleError(ctx context.Context, w http.ResponseWriter, err error) {
	if apiErr, ok := errors.AsType[*apperr.Error](err); ok {
		respond.Error(w, apiErr.Status, apiErr.Message)
		return
	}

	if errors.Is(err, apperr.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "not found")
		return
	}

	slog.ErrorContext(ctx, "unhandled error", slog.Any("error", err))
	respond.Error(w, http.StatusInternalServerError, "internal error")
}
