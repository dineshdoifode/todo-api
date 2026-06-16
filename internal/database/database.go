package database

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // postgres driver for migrate
	_ "github.com/golang-migrate/migrate/v4/source/file"       // file source for migrations
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/dineshdoifode/todo-api/internal/config"
)

// New opens a GORM database connection using the provided config and configures
// the connection pool for production use.
func New(cfg config.DatabaseConfig, log *slog.Logger) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	log.Info("database connection established", "host", cfg.Host, "name", cfg.Name)
	return db, nil
}

// RunMigrations applies all pending up-migrations from the given source path.
func RunMigrations(migrationURL, sourcePath string, log *slog.Logger) error {
	source := fmt.Sprintf("file://%s", sourcePath)

	m, err := migrate.New(source, migrationURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	log.Info("database migrations applied successfully")
	return nil
}
