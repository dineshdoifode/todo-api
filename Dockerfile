# ── Stage 1: Builder ──────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

# Install git for modules that require VCS info
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Cache dependency downloads separately from source compilation
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# Build a statically linked binary with debug info stripped
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o todo-api \
    ./cmd/api

# ── Stage 2: Runtime ──────────────────────────────────────────────────────
FROM scratch

# Bring in timezone data and CA certs from the builder stage
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /build/todo-api /todo-api

# Copy migration files (the app runs migrations on startup)
COPY --from=builder /build/migrations /migrations

EXPOSE 8080

# Run as a non-root user for least-privilege
USER 65534:65534

ENTRYPOINT ["/todo-api"]
