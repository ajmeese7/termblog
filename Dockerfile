# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with CGO enabled (required for SQLite)
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /termblog ./cmd/termblog

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
