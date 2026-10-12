// Package handler provides HTTP handlers for the application.
package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/aarock1234/go-template/pkg/apperr"
	"github.com/aarock1234/go-template/pkg/respond"
)

// Service defines the service methods used by Handler.
type Service interface {
	Example(ctx context.Context) (int32, error)
}

// Handler serves HTTP requests using the service layer.
type Handler struct {
	svc Service
}

// New creates a Handler backed by the given service.
func New(svc Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

type exampleResponse struct {
	Result int32 `json:"result"`
}

// GetExample handles GET /api/example and returns the result of the
// example query.
func (h *Handler) GetExample(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Example(r.Context())
	if err != nil {
		writeError(r.Context(), w, err)
		return
	}

	respond.JSON(w, http.StatusOK, exampleResponse{
		Result: result,
	})
}

// writeError maps err to an HTTP error response. An apperr.Error keeps its
// status and message, sentinel errors map to their conventional statuses,
// and any other error is logged and reported as an internal error.
func writeError(ctx context.Context, w http.ResponseWriter, err error) {
	if appErr, ok := errors.AsType[*apperr.Error](err); ok {
		if appErr.Status >= http.StatusInternalServerError {
			slog.ErrorContext(ctx, "request failed", slog.Any("error", err))
		}

		respond.Error(w, appErr.Status, appErr.Message)

		return
	}

	switch {
	case errors.Is(err, apperr.ErrValidation):
		respond.Error(w, http.StatusBadRequest, apperr.ErrValidation.Error())
	case errors.Is(err, apperr.ErrUnauthorized):
		respond.Error(w, http.StatusUnauthorized, apperr.ErrUnauthorized.Error())
	case errors.Is(err, apperr.ErrNotFound):
		respond.Error(w, http.StatusNotFound, apperr.ErrNotFound.Error())
	case errors.Is(err, apperr.ErrConflict):
		respond.Error(w, http.StatusConflict, apperr.ErrConflict.Error())
	default:
		slog.ErrorContext(ctx, "request failed", slog.Any("error", err))
		respond.Error(w, http.StatusInternalServerError, "internal error")
	}
}
