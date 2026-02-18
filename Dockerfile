# WASM build stage
FROM rust:1.84-alpine AS wasm-builder

RUN apk add --no-cache musl-dev
RUN rustup target add wasm32-unknown-unknown
RUN cargo install trunk --locked

WORKDIR /app/web
COPY web/Cargo.toml web/Cargo.lock* web/Trunk.toml web/index.html ./
COPY web/src ./src

RUN trunk build --release

# Go build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy WASM dist from wasm-builder stage
COPY --from=wasm-builder /app/web/dist ./internal/server/wasm_dist

# Build with CGO enabled (required for SQLite) and FTS5 for full-text search
RUN CGO_ENABLED=1 go build -tags fts5 -ldflags="-s -w" -o /termblog ./cmd/termblog

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN adduser -D -h /app termblog
USER termblog
WORKDIR /app

# Copy binary from builder
COPY --from=builder /termblog /usr/local/bin/termblog

# Create directories for data
RUN mkdir -p /app/content/posts /app/.ssh

# Default config location
VOLUME ["/app/content", "/app/.ssh"]

# Expose ports
EXPOSE 2222 8080

# Default command
CMD ["termblog", "serve"]
