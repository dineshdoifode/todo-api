package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/dineshdoifode/todo-api/internal/dto"
	"github.com/dineshdoifode/todo-api/internal/repository"
	"github.com/dineshdoifode/todo-api/internal/service"
)

// TodoHandler exposes REST endpoints for the todo resource.
type TodoHandler struct {
	svc    service.TodoService
	logger *slog.Logger
}

// NewTodoHandler creates a handler with the given service dependency.
func NewTodoHandler(svc service.TodoService, logger *slog.Logger) *TodoHandler {
	return &TodoHandler{svc: svc, logger: logger}
}

// RegisterRoutes mounts all todo endpoints onto the given chi Router.
func (h *TodoHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/todos", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})
}

// Create handles POST /api/v1/todos
func (h *TodoHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", []string{err.Error()})
		return
	}

	todo, err := h.svc.Create(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Message: "todo created",
		Data:    todo,
	})
}

// GetByID handles GET /api/v1/todos/{id}
func (h *TodoHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	todo, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    todo,
	})
}

// List handles GET /api/v1/todos
func (h *TodoHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := dto.ListFilter{
		IncludeCompleted: strings.EqualFold(r.URL.Query().Get("include_completed"), "true"),
	}

	result, err := h.svc.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    result,
	})
}

// Update handles PUT /api/v1/todos/{id}
func (h *TodoHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req dto.UpdateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", []string{err.Error()})
		return
	}

	todo, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "todo updated",
		Data:    todo,
	})
}

// Delete handles DELETE /api/v1/todos/{id}
func (h *TodoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "todo deleted",
	})
}

// ── helpers ──────────────────────────────────────────────────────────────────

func (h *TodoHandler) handleServiceError(w http.ResponseWriter, err error) {
	var ve *service.ValidationError
	if errors.As(err, &ve) {
		writeError(w, http.StatusUnprocessableEntity, "validation failed", ve.Errors)
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "todo not found", nil)
		return
	}
	h.logger.Error("internal error", "error", err)
	writeError(w, http.StatusInternalServerError, "internal server error", nil)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// At this point headers are already sent; log and move on.
		slog.Error("failed to encode response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string, errs []string) {
	writeJSON(w, status, dto.ErrorResponse{
		Success: false,
		Message: message,
		Errors:  errs,
	})
}
