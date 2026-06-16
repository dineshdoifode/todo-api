package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dineshdoifode/todo-api/internal/dto"
	"github.com/dineshdoifode/todo-api/internal/model"
	"github.com/dineshdoifode/todo-api/internal/repository"
	"github.com/dineshdoifode/todo-api/internal/service"
)

// ── Mock repository ────────────────────────────────────────────────────────

type mockTodoRepo struct{ mock.Mock }

func (m *mockTodoRepo) Create(ctx context.Context, todo *model.Todo) error {
	args := m.Called(ctx, todo)
	return args.Error(0)
}
func (m *mockTodoRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Todo, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Todo), args.Error(1)
}
func (m *mockTodoRepo) List(ctx context.Context, filter dto.ListFilter) ([]model.Todo, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]model.Todo), args.Error(1)
}
func (m *mockTodoRepo) Update(ctx context.Context, todo *model.Todo) error {
	args := m.Called(ctx, todo)
	return args.Error(0)
}
func (m *mockTodoRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// ── Fixtures ───────────────────────────────────────────────────────────────

func newLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func sampleTodo() *model.Todo {
	return &model.Todo{
		ID:        uuid.New(),
		Task:      "Buy groceries",
		DueDate:   time.Now().Add(24 * time.Hour).UTC(),
		Completed: false,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// ── Create ─────────────────────────────────────────────────────────────────

func TestCreate_Success(t *testing.T) {
	repo := new(mockTodoRepo)
	svc := service.NewTodoService(repo, newLogger())

	req := dto.CreateTodoRequest{
		Task:    "Buy groceries",
		DueDate: time.Now().Add(24 * time.Hour),
	}
	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.Todo")).Return(nil)

	resp, err := svc.Create(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "Buy groceries", resp.Task)
	repo.AssertExpectations(t)
}

func TestCreate_EmptyTask(t *testing.T) {
	svc := service.NewTodoService(new(mockTodoRepo), newLogger())

	_, err := svc.Create(context.Background(), dto.CreateTodoRequest{
		Task:    "",
		DueDate: time.Now().Add(time.Hour),
	})

	var ve *service.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Contains(t, ve.Errors, "task is required")
}

func TestCreate_TaskTooLong(t *testing.T) {
	svc := service.NewTodoService(new(mockTodoRepo), newLogger())

	longTask := make([]byte, 501)
	for i := range longTask {
		longTask[i] = 'a'
	}

	_, err := svc.Create(context.Background(), dto.CreateTodoRequest{
		Task:    string(longTask),
		DueDate: time.Now().Add(time.Hour),
	})

	var ve *service.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Contains(t, ve.Errors, "task must not exceed 500 characters")
}

func TestCreate_MissingDueDate(t *testing.T) {
	svc := service.NewTodoService(new(mockTodoRepo), newLogger())

	_, err := svc.Create(context.Background(), dto.CreateTodoRequest{Task: "Something"})

	var ve *service.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Contains(t, ve.Errors, "due_date is required")
}

func TestCreate_RepoError(t *testing.T) {
	repo := new(mockTodoRepo)
	svc := service.NewTodoService(repo, newLogger())

	repo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db down"))

	_, err := svc.Create(context.Background(), dto.CreateTodoRequest{
		Task:    "Something",
		DueDate: time.Now().Add(time.Hour),
	})
	require.Error(t, err)
}

// ── GetByID ────────────────────────────────────────────────────────────────

func TestGetByID_Success(t *testing.T) {
	repo := new(mockTodoRepo)
	svc := service.NewTodoService(repo, newLogger())
	todo := sampleTodo()

	repo.On("GetByID", mock.Anything, todo.ID).Return(todo, nil)

	resp, err := svc.GetByID(context.Background(), todo.ID.String())
	require.NoError(t, err)
	assert.Equal(t, todo.ID, resp.ID)
}

func TestGetByID_InvalidUUID(t *testing.T) {
	svc := service.NewTodoService(new(mockTodoRepo), newLogger())

	_, err := svc.GetByID(context.Background(), "not-a-uuid")

	var ve *service.ValidationError
	require.ErrorAs(t, err, &ve)
}

func TestGetByID_NotFound(t *testing.T) {
	repo := new(mockTodoRepo)
	svc := service.NewTodoService(repo, newLogger())
	id := uuid.New()

	repo.On("GetByID", mock.Anything, id).Return(nil, repository.ErrNotFound)

	_, err := svc.GetByID(context.Background(), id.String())
	require.ErrorIs(t, err, repository.ErrNotFound)
}

// ── List ───────────────────────────────────────────────────────────────────

func TestList_ExcludesCompleted(t *testing.T) {
	repo := new(mockTodoRepo)
	svc := service.NewTodoService(repo, newLogger())

	filter := dto.ListFilter{IncludeCompleted: false}
	repo.On("List", mock.Anything, filter).Return([]model.Todo{*sampleTodo()}, nil)

	resp, err := svc.List(context.Background(), filter)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
}

func TestList_IncludeCompleted(t *testing.T) {
	repo := new(mockTodoRepo)
	svc := service.NewTodoService(repo, newLogger())

	completed := sampleTodo()
	completed.Completed = true

	filter := dto.ListFilter{IncludeCompleted: true}
	repo.On("List", mock.Anything, filter).Return([]model.Todo{*sampleTodo(), *completed}, nil)

	resp, err := svc.List(context.Background(), filter)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
}

// ── Update ─────────────────────────────────────────────────────────────────

func TestUpdate_Success(t *testing.T) {
	repo := new(mockTodoRepo)
	svc := service.NewTodoService(repo, newLogger())
	todo := sampleTodo()

	newTask := "Updated task"
	req := dto.UpdateTodoRequest{Task: &newTask}

	repo.On("GetByID", mock.Anything, todo.ID).Return(todo, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.Todo")).Return(nil)

	resp, err := svc.Update(context.Background(), todo.ID.String(), req)
	require.NoError(t, err)
	assert.Equal(t, "Updated task", resp.Task)
}

func TestUpdate_NotFound(t *testing.T) {
	repo := new(mockTodoRepo)
	svc := service.NewTodoService(repo, newLogger())
	id := uuid.New()

	repo.On("GetByID", mock.Anything, id).Return(nil, repository.ErrNotFound)

	_, err := svc.Update(context.Background(), id.String(), dto.UpdateTodoRequest{})
	require.ErrorIs(t, err, repository.ErrNotFound)
}

func TestUpdate_EmptyTaskValidation(t *testing.T) {
	repo := new(mockTodoRepo)
	svc := service.NewTodoService(repo, newLogger())
	todo := sampleTodo()

	empty := ""
	req := dto.UpdateTodoRequest{Task: &empty}

	repo.On("GetByID", mock.Anything, todo.ID).Return(todo, nil)

	_, err := svc.Update(context.Background(), todo.ID.String(), req)
	var ve *service.ValidationError
	require.ErrorAs(t, err, &ve)
}

// ── Delete ─────────────────────────────────────────────────────────────────

func TestDelete_Success(t *testing.T) {
	repo := new(mockTodoRepo)
	svc := service.NewTodoService(repo, newLogger())
	id := uuid.New()

	repo.On("Delete", mock.Anything, id).Return(nil)

	err := svc.Delete(context.Background(), id.String())
	require.NoError(t, err)
}

func TestDelete_InvalidUUID(t *testing.T) {
	svc := service.NewTodoService(new(mockTodoRepo), newLogger())

	err := svc.Delete(context.Background(), "bad-id")
	var ve *service.ValidationError
	require.ErrorAs(t, err, &ve)
}

func TestDelete_NotFound(t *testing.T) {
	repo := new(mockTodoRepo)
	svc := service.NewTodoService(repo, newLogger())
	id := uuid.New()

	repo.On("Delete", mock.Anything, id).Return(repository.ErrNotFound)

	err := svc.Delete(context.Background(), id.String())
	require.ErrorIs(t, err, repository.ErrNotFound)
}
