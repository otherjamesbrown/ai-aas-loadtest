# Multi-stage build for minimal image size
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -o load-test-worker \
    ./cmd/load-test-worker

# Final minimal image
FROM alpine:3.19

# Add ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 loadtest && \
    adduser -D -u 1000 -G loadtest loadtest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/load-test-worker .

# Copy example configs for reference
COPY configs/examples /app/configs/examples

# Set ownership
RUN chown -R loadtest:loadtest /app

# Switch to non-root user
USER loadtest

# Set entrypoint
ENTRYPOINT ["/app/load-test-worker"]

# Default command shows help
CMD ["--help"]
