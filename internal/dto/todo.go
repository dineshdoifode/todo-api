package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateTodoRequest is the body accepted by POST /api/v1/todos.
type CreateTodoRequest struct {
	DueDate time.Time `json:"due_date"`
	Task    string    `json:"task"`
}

// UpdateTodoRequest is the body accepted by PUT /api/v1/todos/{id}.
// All fields are optional; only non-zero values are applied.
type UpdateTodoRequest struct {
	Task      *string    `json:"task"`
	DueDate   *time.Time `json:"due_date"`
	Completed *bool      `json:"completed"`
}

// TodoResponse is returned for single-item endpoints.
type TodoResponse struct {
	DueDate   time.Time `json:"due_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Task      string    `json:"task"`
	ID        uuid.UUID `json:"id"`
	Completed bool      `json:"completed"`
}

// ListTodosResponse is the envelope for the list endpoint.
type ListTodosResponse struct {
	Data  []TodoResponse `json:"data"`
	Total int            `json:"total"`
}

// SuccessResponse wraps any successful payload.
type SuccessResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Success bool        `json:"success"`
}

// ErrorResponse is returned on any API error.
type ErrorResponse struct {
	Message string   `json:"message"`
	Errors  []string `json:"errors,omitempty"`
	Success bool     `json:"success"`
}

// ListFilter carries query-parameter filters for the list endpoint.
type ListFilter struct {
	IncludeCompleted bool
}
