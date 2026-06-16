package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dineshdoifode/todo-api/internal/dto"
	"github.com/dineshdoifode/todo-api/internal/model"
	"github.com/dineshdoifode/todo-api/internal/repository"
)

// ValidationError carries field-level validation failures back to the handler.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return "validation failed: " + strings.Join(e.Errors, "; ")
}

// TodoService defines the business-logic contract consumed by handlers.
type TodoService interface {
	Create(ctx context.Context, req dto.CreateTodoRequest) (*dto.TodoResponse, error)
	GetByID(ctx context.Context, id string) (*dto.TodoResponse, error)
	List(ctx context.Context, filter dto.ListFilter) (*dto.ListTodosResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateTodoRequest) (*dto.TodoResponse, error)
	Delete(ctx context.Context, id string) error
}

type todoService struct {
	repo   repository.TodoRepository
	logger *slog.Logger
}

// NewTodoService wires a TodoService with the given repository and logger.
func NewTodoService(repo repository.TodoRepository, logger *slog.Logger) TodoService {
	return &todoService{repo: repo, logger: logger}
}

// Create validates the request and persists a new todo.
func (s *todoService) Create(ctx context.Context, req dto.CreateTodoRequest) (*dto.TodoResponse, error) {
	if errs := validateCreateRequest(req); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	todo := &model.Todo{
		Task:    strings.TrimSpace(req.Task),
		DueDate: req.DueDate.UTC(),
	}

	if err := s.repo.Create(ctx, todo); err != nil {
		s.logger.Error("failed to create todo", "error", err)
		return nil, fmt.Errorf("create todo: %w", err)
	}

	s.logger.Info("todo created", "id", todo.ID)
	return toResponse(todo), nil
}

// GetByID retrieves a single todo, returning a user-friendly error on bad UUID or missing record.
func (s *todoService) GetByID(ctx context.Context, rawID string) (*dto.TodoResponse, error) {
	id, err := parseUUID(rawID)
	if err != nil {
		return nil, &ValidationError{Errors: []string{"invalid uuid format"}}
	}

	todo, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, repository.ErrNotFound
		}
		s.logger.Error("failed to get todo", "id", id, "error", err)
		return nil, fmt.Errorf("get todo: %w", err)
	}

	return toResponse(todo), nil
}

// List applies filters and delegates to the repository.
func (s *todoService) List(ctx context.Context, filter dto.ListFilter) (*dto.ListTodosResponse, error) {
	todos, err := s.repo.List(ctx, filter)
	if err != nil {
		s.logger.Error("failed to list todos", "error", err)
		return nil, fmt.Errorf("list todos: %w", err)
	}

	responses := make([]dto.TodoResponse, 0, len(todos))
	for i := range todos {
		responses = append(responses, *toResponse(&todos[i]))
	}

	return &dto.ListTodosResponse{Data: responses, Total: len(responses)}, nil
}

// Update applies partial updates, validating any supplied fields.
func (s *todoService) Update(ctx context.Context, rawID string, req dto.UpdateTodoRequest) (*dto.TodoResponse, error) {
	id, err := parseUUID(rawID)
	if err != nil {
		return nil, &ValidationError{Errors: []string{"invalid uuid format"}}
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("get todo for update: %w", err)
	}

	if errs := validateUpdateRequest(req); len(errs) > 0 {
		return nil, &ValidationError{Errors: errs}
	}

	if req.Task != nil {
		existing.Task = strings.TrimSpace(*req.Task)
	}
	if req.DueDate != nil {
		existing.DueDate = req.DueDate.UTC()
	}
	if req.Completed != nil {
		existing.Completed = *req.Completed
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update todo", "id", id, "error", err)
		return nil, fmt.Errorf("update todo: %w", err)
	}

	s.logger.Info("todo updated", "id", id)
	return toResponse(existing), nil
}

// Delete removes a todo by ID.
func (s *todoService) Delete(ctx context.Context, rawID string) error {
	id, err := parseUUID(rawID)
	if err != nil {
		return &ValidationError{Errors: []string{"invalid uuid format"}}
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return repository.ErrNotFound
		}
		s.logger.Error("failed to delete todo", "id", id, "error", err)
		return fmt.Errorf("delete todo: %w", err)
	}

	s.logger.Info("todo deleted", "id", id)
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func validateCreateRequest(req dto.CreateTodoRequest) []string {
	var errs []string
	task := strings.TrimSpace(req.Task)
	if task == "" {
		errs = append(errs, "task is required")
	} else if len(task) > 500 {
		errs = append(errs, "task must not exceed 500 characters")
	}
	if req.DueDate.IsZero() {
		errs = append(errs, "due_date is required")
	}
	return errs
}

func validateUpdateRequest(req dto.UpdateTodoRequest) []string {
	var errs []string
	if req.Task != nil {
		task := strings.TrimSpace(*req.Task)
		if task == "" {
			errs = append(errs, "task must not be empty")
		} else if len(task) > 500 {
			errs = append(errs, "task must not exceed 500 characters")
		}
	}
	if req.DueDate != nil && req.DueDate.IsZero() {
		errs = append(errs, "due_date must not be zero")
	}
	return errs
}

func parseUUID(raw string) (uuid.UUID, error) {
	return uuid.Parse(raw)
}

func toResponse(t *model.Todo) *dto.TodoResponse {
	return &dto.TodoResponse{
		ID:        t.ID,
		Task:      t.Task,
		DueDate:   t.DueDate,
		Completed: t.Completed,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}
