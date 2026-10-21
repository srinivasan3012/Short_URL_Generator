# syntax=docker/dockerfile:1

# -------------------------------------------------------------
# Stage 1: Build binary
# -------------------------------------------------------------
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install ca-certificates and git for downloading dependencies
RUN apk add --no-cache ca-certificates git tzdata

# Leverage Docker cache layers by copying module files first
COPY go.mod go.sum ./
RUN go mod download

# Copy source tree
COPY . .

# Compile optimized static binary with symbols stripped (-s -w) and path trimming
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /bin/api \
    ./cmd/api

# Create non-root user
RUN addgroup -S appgroup -g 10001 && \
    adduser -S -D -H -u 10001 -s /sbin/nologin -G appgroup appuser

# -------------------------------------------------------------
# Stage 2: Minimal Production Runtime
# -------------------------------------------------------------
FROM alpine:3.20

# Install runtime dependencies for healthchecks and TLS certificates
RUN apk add --no-cache ca-certificates tzdata

# Copy non-root user from builder
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy compiled binary
COPY --from=builder /bin/api /bin/api

# Switch to non-root user
USER appuser

EXPOSE 8080

# Container liveness check
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/bin/api"]
