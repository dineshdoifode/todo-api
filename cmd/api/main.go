// Package main is the entry point for the Todo API service.
//
//	@title						Todo API
//	@version					1.0
//	@description				A production-ready RESTful Todo service built with Go, Chi, GORM, and PostgreSQL.
//	@description				Demonstrates clean architecture with repository, service, and handler layers.
//
//	@contact.name				Dinesh Doifode
//	@contact.email				dineshdoifode@gmail.com
//	@contact.url				https://linkedin.com/in/dineshdoifode
//
//	@license.name				MIT
//	@license.url				https://opensource.org/licenses/MIT
//
//	@host						localhost:8080
//	@BasePath					/
//	@schemes					http https
//
//	@tag.name					todos
//	@tag.description			Operations for managing to-do items
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/dineshdoifode/todo-api/internal/config"
	"github.com/dineshdoifode/todo-api/internal/database"
	"github.com/dineshdoifode/todo-api/internal/handler"
	"github.com/dineshdoifode/todo-api/internal/logger"
	"github.com/dineshdoifode/todo-api/internal/middleware"
	"github.com/dineshdoifode/todo-api/internal/repository"
	"github.com/dineshdoifode/todo-api/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// ── Configuration ──────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// ── Logger ─────────────────────────────────────────────────────────────
	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	log.Info("starting todo-api", "version", "1.0.0")

	// ── Database ───────────────────────────────────────────────────────────
	db, err := database.New(cfg.Database, log)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}

	if err := database.RunMigrations(cfg.Database.MigrationURL(), "migrations", log); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	// ── Dependency Injection ───────────────────────────────────────────────
	todoRepo := repository.NewTodoRepository(db)
	todoSvc := service.NewTodoService(todoRepo, log)
	todoHandler := handler.NewTodoHandler(todoSvc, log)

	// ── Router ─────────────────────────────────────────────────────────────
	r := chi.NewRouter()

	// Core middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Recovery(log))
	r.Use(middleware.Logger(log))
	r.Use(chimiddleware.Compress(5))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health check — useful for Docker/K8s liveness probes
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			log.Error("health check write failed", "error", err)
		}
	})
	// API routes
	todoHandler.RegisterRoutes(r)

	// Swagger UI — served at /swagger/index.html
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// ── HTTP Server ────────────────────────────────────────────────────────
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Graceful Shutdown ──────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info("http server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			quit <- os.Interrupt
		}
	}()

	<-quit
	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	sqlDB, _ := db.DB()
	if err := sqlDB.Close(); err != nil {
		log.Error("error closing database", "error", err)
	}

	log.Info("server stopped gracefully")
	return nil
}
