package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateTodoRequest is the body accepted by POST /api/v1/todos.
type CreateTodoRequest struct {
	Task    string    `json:"task"`
	DueDate time.Time `json:"due_date"`
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
	ID        uuid.UUID `json:"id"`
	Task      string    `json:"task"`
	DueDate   time.Time `json:"due_date"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListTodosResponse is the envelope for the list endpoint.
type ListTodosResponse struct {
	Data  []TodoResponse `json:"data"`
	Total int            `json:"total"`
}

// SuccessResponse wraps any successful payload.
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse is returned on any API error.
type ErrorResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Errors  []string `json:"errors,omitempty"`
}

// ListFilter carries query-parameter filters for the list endpoint.
type ListFilter struct {
	IncludeCompleted bool
}
