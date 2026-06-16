# Todo API

A production-ready RESTful Todo service built in **Go 1.24**, demonstrating clean architecture, layered dependency injection, structured logging, database migrations, and a full CI/CD pipeline.

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     HTTP Client                         │
└──────────────────────┬──────────────────────────────────┘
                       │ REST (JSON)
┌──────────────────────▼──────────────────────────────────┐
│               Chi Router + Middleware                   │
│        (RequestID · Logger · Recovery · CORS)           │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│                  Handler Layer                          │
│   Decode request → call service → encode response      │
└──────────────────────┬──────────────────────────────────┘
                       │ interface
┌──────────────────────▼──────────────────────────────────┐
│                  Service Layer                          │
│       Validation · Business rules · Orchestration      │
└──────────────────────┬──────────────────────────────────┘
                       │ interface
┌──────────────────────▼──────────────────────────────────┐
│               Repository Layer                          │
│        GORM ▸ PostgreSQL (golang-migrate)               │
└─────────────────────────────────────────────────────────┘
```

### Project Structure

```
todo-api/
├── cmd/
│   └── api/
│       └── main.go              # Entry point, DI wiring, graceful shutdown
├── internal/
│   ├── config/        config.go # Env-var driven configuration
│   ├── logger/        logger.go # slog factory (JSON / text)
│   ├── database/      database.go # GORM connection + migrate runner
│   ├── model/         todo.go   # GORM domain model
│   ├── dto/           todo.go   # Request / response shapes
│   ├── repository/    todo_repository.go + test
│   ├── service/       todo_service.go + test
│   ├── handler/       todo_handler.go + test
│   └── middleware/    middleware.go
├── migrations/
│   ├── 000001_create_todos_table.up.sql
│   └── 000001_create_todos_table.down.sql
├── .github/
│   └── workflows/go.yml         # CI: lint → test → build → push image
├── Dockerfile                   # Multi-stage scratch image
├── docker-compose.yml
├── Makefile
├── .env                         # Local dev defaults (not committed in prod)
└── README.md
```

---

## Prerequisites

| Tool | Version |
|------|---------|
| Go | 1.24+ |
| Docker | 24+ |
| Docker Compose | v2+ |
| [golang-migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) | v4+ |

---

## Quick Start (Docker Compose)

```bash
# 1. Clone
git clone https://github.com/dineshdoifode/todo-api.git
cd todo-api

# 2. Copy env file (edit values as needed)
cp .env .env.local   # or just use .env directly

# 3. Start everything (API + Postgres)
docker compose up --build

# API is now available at http://localhost:8080
```

Migrations run automatically on startup.

---

## Local Development

```bash
# Start only Postgres
docker compose up -d postgres

# Run the API locally
make run

# Or equivalently
go run ./cmd/api
```

---

## Database Migration Commands

```bash
# Apply all pending migrations
make migrate-up

# Revert last migration
make migrate-down

# Drop everything (destructive)
make migrate-drop

# Run directly with the CLI
migrate -path migrations \
        -database "postgres://postgres:postgres@localhost:5432/tododb?sslmode=disable" \
        up
```

---

## Running Tests

```bash
# Unit tests (no database required)
make test

# Unit tests + coverage report (coverage.html opens in browser)
make test-coverage

# Integration tests (requires running Postgres on :5432)
make test-integration

# Shortcut for a single package
go test -v ./internal/service/...
go test -v ./internal/handler/...
go test -v ./internal/repository/...
```

---

## API Reference

Base URL: `http://localhost:8080`

### Response Envelope

**Success**
```json
{ "success": true, "message": "...", "data": { ... } }
```

**Error**
```json
{ "success": false, "message": "validation failed", "errors": ["task is required"] }
```

---

### Endpoints

#### `POST /api/v1/todos` — Create a todo

```bash
curl -s -X POST http://localhost:8080/api/v1/todos \
  -H "Content-Type: application/json" \
  -d '{
    "task": "Write unit tests",
    "due_date": "2025-12-31T18:00:00Z"
  }' | jq
```

**Response 201**
```json
{
  "success": true,
  "message": "todo created",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "task": "Write unit tests",
    "due_date": "2025-12-31T18:00:00Z",
    "completed": false,
    "created_at": "2025-06-01T10:00:00Z",
    "updated_at": "2025-06-01T10:00:00Z"
  }
}
```

---

#### `GET /api/v1/todos/{id}` — Get a todo

```bash
curl -s http://localhost:8080/api/v1/todos/550e8400-e29b-41d4-a716-446655440000 | jq
```

---

#### `GET /api/v1/todos` — List todos

Sorted by `due_date` ascending. Completed tasks excluded by default.

```bash
# Pending only (default)
curl -s http://localhost:8080/api/v1/todos | jq

# Include completed
curl -s "http://localhost:8080/api/v1/todos?include_completed=true" | jq
```

**Response 200**
```json
{
  "success": true,
  "data": {
    "data": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "task": "Write unit tests",
        "due_date": "2025-12-31T18:00:00Z",
        "completed": false,
        "created_at": "2025-06-01T10:00:00Z",
        "updated_at": "2025-06-01T10:00:00Z"
      }
    ],
    "total": 1
  }
}
```

---

#### `PUT /api/v1/todos/{id}` — Update a todo

All fields are optional. Only supplied fields are modified.

```bash
curl -s -X PUT http://localhost:8080/api/v1/todos/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{"completed": true}' | jq

# Update task text and due date
curl -s -X PUT http://localhost:8080/api/v1/todos/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json" \
  -d '{
    "task": "Write AND run unit tests",
    "due_date": "2025-11-30T12:00:00Z"
  }' | jq
```

---

#### `DELETE /api/v1/todos/{id}` — Delete a todo

```bash
curl -s -X DELETE http://localhost:8080/api/v1/todos/550e8400-e29b-41d4-a716-446655440000 | jq
```

**Response 200**
```json
{ "success": true, "message": "todo deleted" }
```

---

### HTTP Status Codes

| Status | Meaning |
|--------|---------|
| `200 OK` | Successful read / update / delete |
| `201 Created` | New todo created |
| `400 Bad Request` | Malformed JSON body |
| `404 Not Found` | Todo does not exist |
| `422 Unprocessable Entity` | Validation failure |
| `500 Internal Server Error` | Unexpected server error |

---

### Health Check

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_HOST` | `0.0.0.0` | Bind address |
| `SERVER_PORT` | `8080` | Listen port |
| `SERVER_READ_TIMEOUT` | `30` | Read timeout (seconds) |
| `SERVER_WRITE_TIMEOUT` | `30` | Write timeout (seconds) |
| `DB_HOST` | `localhost` | Postgres host |
| `DB_PORT` | `5432` | Postgres port |
| `DB_USER` | `postgres` | Postgres user |
| `DB_PASSWORD` | `postgres` | Postgres password |
| `DB_NAME` | `tododb` | Database name |
| `DB_SSLMODE` | `disable` | Postgres SSL mode |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | `json` or `text` |

---

## CI/CD Pipeline

The GitHub Actions workflow (`.github/workflows/go.yml`) runs on every push and PR to `main`/`develop`:

```
push / PR
   │
   ├─► Lint & Vet  (gofmt + go vet + golangci-lint)
   │
   ├─► Unit Tests  (service + handler, no DB needed)
   │        └── coverage gate ≥ 80%
   │
   ├─► Integration Tests  (repository, real Postgres service container)
   │
   ├─► Build Binary
   │
   └─► [main only] Push Docker image → GitHub Container Registry
```

---

## Docker Commands

```bash
# Full stack up (build + start)
docker compose up --build

# Detached mode
docker compose up --build -d

# Tail API logs
docker compose logs -f todo-api

# Stop and remove containers + volumes
docker compose down -v

# Rebuild API only
docker compose up --build todo-api
```

---

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Chi router** | Lightweight, idiomatic, no global state, excellent middleware support |
| **GORM** | Reduces boilerplate for CRUD; Repository pattern keeps it swappable |
| **golang-migrate** | SQL-first migrations — readable, version-controlled, rollback-safe |
| **slog** | Standard library; structured JSON logs ready for Loki/CloudWatch |
| **Interface-driven DI** | Every layer depends on an interface, not a concrete type — 100% mockable |
| **Scratch image** | Minimal attack surface; binary + certs only |
| **Graceful shutdown** | SIGTERM drains in-flight requests before closing the DB pool |

---

## Author

**Dinesh Doifode**  
Senior Backend & IoT Engineer  
📧 dineshdoifode@gmail.com  
🔗 [linkedin.com/in/dineshdoifode](https://linkedin.com/in/dineshdoifode)
