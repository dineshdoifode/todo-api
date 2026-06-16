.PHONY: all build run test test-coverage lint fmt vet docker-up docker-down migrate-up migrate-down clean

BINARY      := todo-api
CMD_DIR     := ./cmd/api
MIGRATE_URL ?= postgres://postgres:postgres@localhost:5432/tododb?sslmode=disable
MIGRATIONS  := migrations

all: lint test build

## build: compile the binary
build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BINARY) $(CMD_DIR)

## run: run the API locally (requires a running Postgres instance)
run:
	go run $(CMD_DIR)

## test: run all unit tests
test:
	go test -v -race -count=1 ./internal/service/... ./internal/handler/...

## test-integration: run all tests including repository integration tests
test-integration:
	go test -v -race -count=1 ./...

## test-coverage: generate coverage report
test-coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic \
		./internal/service/... ./internal/handler/...
	go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total

## lint: run linters
lint:
	golangci-lint run ./...

## fmt: format all Go source files
fmt:
	gofmt -w .
	goimports -w .

## vet: run go vet
vet:
	go vet ./...

## docker-up: start all services with Docker Compose
docker-up:
	docker compose up --build -d

## docker-down: stop and remove all containers
docker-down:
	docker compose down -v

## docker-logs: tail API logs
docker-logs:
	docker compose logs -f todo-api

## migrate-up: apply pending migrations
migrate-up:
	migrate -path $(MIGRATIONS) -database "$(MIGRATE_URL)" up

## migrate-down: revert the last migration
migrate-down:
	migrate -path $(MIGRATIONS) -database "$(MIGRATE_URL)" down 1

## migrate-drop: drop all migrations (danger!)
migrate-drop:
	migrate -path $(MIGRATIONS) -database "$(MIGRATE_URL)" drop -f

## clean: remove build artifacts
clean:
	rm -f $(BINARY) coverage.out coverage.html
