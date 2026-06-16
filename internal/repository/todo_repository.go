package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/dineshdoifode/todo-api/internal/dto"
	"github.com/dineshdoifode/todo-api/internal/model"
)

// ErrNotFound is returned when a requested todo does not exist.
var ErrNotFound = errors.New("todo not found")

// TodoRepository defines the persistence contract for todo items.
// Coding against an interface allows easy mocking in tests and swapping
// the underlying store (e.g., moving from PostgreSQL to a cache layer).
type TodoRepository interface {
	Create(ctx context.Context, todo *model.Todo) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Todo, error)
	List(ctx context.Context, filter dto.ListFilter) ([]model.Todo, error)
	Update(ctx context.Context, todo *model.Todo) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// gormTodoRepository is the PostgreSQL-backed implementation of TodoRepository.
type gormTodoRepository struct {
	db *gorm.DB
}

// NewTodoRepository creates a production-ready repository backed by GORM/Postgres.
func NewTodoRepository(db *gorm.DB) TodoRepository {
	return &gormTodoRepository{db: db}
}

// Create inserts a new todo record.
func (r *gormTodoRepository) Create(ctx context.Context, todo *model.Todo) error {
	if err := r.db.WithContext(ctx).Create(todo).Error; err != nil {
		return fmt.Errorf("repository.Create: %w", err)
	}
	return nil
}

// GetByID fetches a single todo by its primary key UUID.
func (r *gormTodoRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Todo, error) {
	var todo model.Todo
	err := r.db.WithContext(ctx).First(&todo, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetByID: %w", err)
	}
	return &todo, nil
}

// List returns all todos ordered by due_date ascending.
// When filter.IncludeCompleted is false, completed tasks are excluded.
func (r *gormTodoRepository) List(ctx context.Context, filter dto.ListFilter) ([]model.Todo, error) {
	var todos []model.Todo
	q := r.db.WithContext(ctx).Order("due_date ASC")

	if !filter.IncludeCompleted {
		q = q.Where("completed = ?", false)
	}

	if err := q.Find(&todos).Error; err != nil {
		return nil, fmt.Errorf("repository.List: %w", err)
	}
	return todos, nil
}

// Update persists changes to an existing todo. Only non-zero columns are updated
// when using Save, which performs a full-record update — intentional here because
// the service layer controls partial-update logic before calling this method.
func (r *gormTodoRepository) Update(ctx context.Context, todo *model.Todo) error {
	result := r.db.WithContext(ctx).Save(todo)
	if result.Error != nil {
		return fmt.Errorf("repository.Update: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a todo by its UUID and returns ErrNotFound when it does not exist.
func (r *gormTodoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&model.Todo{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("repository.Delete: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
