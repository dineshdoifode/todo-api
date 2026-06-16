package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/dineshdoifode/todo-api/internal/dto"
	"github.com/dineshdoifode/todo-api/internal/model"
	"github.com/dineshdoifode/todo-api/internal/repository"
)

// openTestDB opens a real Postgres connection pointed at the test database.
// It is skipped automatically when the DSN environment variable is absent,
// keeping CI clean when Postgres is not available.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=tododb_test sslmode=disable TimeZone=UTC"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("skipping repository integration tests (no postgres): %v", err)
	}

	// Auto-migrate for test isolation
	err = db.AutoMigrate(&model.Todo{})
	require.NoError(t, err)

	// Truncate between tests
	t.Cleanup(func() {
		db.Exec("TRUNCATE TABLE todos")
	})

	return db
}

func TestRepository_CreateAndGet(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewTodoRepository(db)

	todo := &model.Todo{
		Task:    "Integration test task",
		DueDate: time.Now().Add(24 * time.Hour).UTC(),
	}

	err := repo.Create(context.Background(), todo)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, todo.ID)

	fetched, err := repo.GetByID(context.Background(), todo.ID)
	require.NoError(t, err)
	assert.Equal(t, todo.Task, fetched.Task)
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewTodoRepository(db)

	_, err := repo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestRepository_List_ExcludesCompleted(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewTodoRepository(db)

	pending := &model.Todo{Task: "Pending", DueDate: time.Now().Add(time.Hour).UTC(), Completed: false}
	done := &model.Todo{Task: "Done", DueDate: time.Now().Add(2 * time.Hour).UTC(), Completed: true}

	require.NoError(t, repo.Create(context.Background(), pending))
	require.NoError(t, repo.Create(context.Background(), done))

	todos, err := repo.List(context.Background(), dto.ListFilter{IncludeCompleted: false})
	require.NoError(t, err)

	for _, td := range todos {
		assert.False(t, td.Completed, "completed todos should be excluded")
	}
}

func TestRepository_List_IncludesCompleted(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewTodoRepository(db)

	pending := &model.Todo{Task: "Pending", DueDate: time.Now().Add(time.Hour).UTC(), Completed: false}
	done := &model.Todo{Task: "Done", DueDate: time.Now().Add(2 * time.Hour).UTC(), Completed: true}

	require.NoError(t, repo.Create(context.Background(), pending))
	require.NoError(t, repo.Create(context.Background(), done))

	todos, err := repo.List(context.Background(), dto.ListFilter{IncludeCompleted: true})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(todos), 2)
}

func TestRepository_Update(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewTodoRepository(db)

	todo := &model.Todo{Task: "Original", DueDate: time.Now().Add(time.Hour).UTC()}
	require.NoError(t, repo.Create(context.Background(), todo))

	todo.Task = "Updated"
	todo.Completed = true
	require.NoError(t, repo.Update(context.Background(), todo))

	fetched, err := repo.GetByID(context.Background(), todo.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", fetched.Task)
	assert.True(t, fetched.Completed)
}

func TestRepository_Delete(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewTodoRepository(db)

	todo := &model.Todo{Task: "To delete", DueDate: time.Now().Add(time.Hour).UTC()}
	require.NoError(t, repo.Create(context.Background(), todo))

	require.NoError(t, repo.Delete(context.Background(), todo.ID))

	_, err := repo.GetByID(context.Background(), todo.ID)
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestRepository_Delete_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := repository.NewTodoRepository(db)

	err := repo.Delete(context.Background(), uuid.New())
	assert.ErrorIs(t, err, repository.ErrNotFound)
}
