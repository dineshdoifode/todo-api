package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dineshdoifode/todo-api/internal/dto"
	"github.com/dineshdoifode/todo-api/internal/handler"
	"github.com/dineshdoifode/todo-api/internal/repository"
	"github.com/dineshdoifode/todo-api/internal/service"
)

// ── Mock service ───────────────────────────────────────────────────────────

type mockTodoService struct{ mock.Mock }

func (m *mockTodoService) Create(ctx context.Context, req dto.CreateTodoRequest) (*dto.TodoResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TodoResponse), args.Error(1)
}
func (m *mockTodoService) GetByID(ctx context.Context, id string) (*dto.TodoResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TodoResponse), args.Error(1)
}
func (m *mockTodoService) List(ctx context.Context, filter dto.ListFilter) (*dto.ListTodosResponse, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ListTodosResponse), args.Error(1)
}
func (m *mockTodoService) Update(ctx context.Context, id string, req dto.UpdateTodoRequest) (*dto.TodoResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TodoResponse), args.Error(1)
}
func (m *mockTodoService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// ── Fixtures ───────────────────────────────────────────────────────────────

func newLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func sampleResponse() *dto.TodoResponse {
	return &dto.TodoResponse{
		ID:        uuid.New(),
		Task:      "Buy groceries",
		DueDate:   time.Now().Add(24 * time.Hour),
		Completed: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func newRouter(svc service.TodoService) http.Handler {
	r := chi.NewRouter()
	handler.NewTodoHandler(svc, newLogger()).RegisterRoutes(r)
	return r
}

// ── Create ─────────────────────────────────────────────────────────────────

func TestCreate_201(t *testing.T) {
	svc := new(mockTodoService)
	resp := sampleResponse()

	svc.On("Create", mock.Anything, mock.AnythingOfType("dto.CreateTodoRequest")).Return(resp, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"task":     "Buy groceries",
		"due_date": time.Now().Add(24 * time.Hour),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newRouter(svc).ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var out dto.SuccessResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&out))
	assert.True(t, out.Success)
}

func TestCreate_ValidationError_422(t *testing.T) {
	svc := new(mockTodoService)
	svc.On("Create", mock.Anything, mock.Anything).
		Return(nil, &service.ValidationError{Errors: []string{"task is required"}})

	body, _ := json.Marshal(map[string]interface{}{"task": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreate_BadJSON_400(t *testing.T) {
	svc := new(mockTodoService)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── GetByID ────────────────────────────────────────────────────────────────

func TestGetByID_200(t *testing.T) {
	svc := new(mockTodoService)
	resp := sampleResponse()
	svc.On("GetByID", mock.Anything, resp.ID.String()).Return(resp, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/"+resp.ID.String(), nil)
	w := httptest.NewRecorder()

	newRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetByID_NotFound_404(t *testing.T) {
	svc := new(mockTodoService)
	id := uuid.New().String()
	svc.On("GetByID", mock.Anything, id).Return(nil, repository.ErrNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/"+id, nil)
	w := httptest.NewRecorder()

	newRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── List ───────────────────────────────────────────────────────────────────

func TestList_200_ExcludesCompleted(t *testing.T) {
	svc := new(mockTodoService)
	svc.On("List", mock.Anything, dto.ListFilter{IncludeCompleted: false}).
		Return(&dto.ListTodosResponse{Data: []dto.TodoResponse{*sampleResponse()}, Total: 1}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	w := httptest.NewRecorder()

	newRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestList_200_IncludeCompleted(t *testing.T) {
	svc := new(mockTodoService)
	svc.On("List", mock.Anything, dto.ListFilter{IncludeCompleted: true}).
		Return(&dto.ListTodosResponse{Data: []dto.TodoResponse{}, Total: 0}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos?include_completed=true", nil)
	w := httptest.NewRecorder()

	newRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── Update ─────────────────────────────────────────────────────────────────

func TestUpdate_200(t *testing.T) {
	svc := new(mockTodoService)
	resp := sampleResponse()
	svc.On("Update", mock.Anything, resp.ID.String(), mock.AnythingOfType("dto.UpdateTodoRequest")).
		Return(resp, nil)

	newTask := "Updated"
	body, _ := json.Marshal(dto.UpdateTodoRequest{Task: &newTask})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todos/"+resp.ID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdate_NotFound_404(t *testing.T) {
	svc := new(mockTodoService)
	id := uuid.New().String()
	svc.On("Update", mock.Anything, id, mock.Anything).Return(nil, repository.ErrNotFound)

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todos/"+id, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── Delete ─────────────────────────────────────────────────────────────────

func TestDelete_200(t *testing.T) {
	svc := new(mockTodoService)
	id := uuid.New().String()
	svc.On("Delete", mock.Anything, id).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/todos/"+id, nil)
	w := httptest.NewRecorder()

	newRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDelete_NotFound_404(t *testing.T) {
	svc := new(mockTodoService)
	id := uuid.New().String()
	svc.On("Delete", mock.Anything, id).Return(repository.ErrNotFound)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/todos/"+id, nil)
	w := httptest.NewRecorder()

	newRouter(svc).ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
