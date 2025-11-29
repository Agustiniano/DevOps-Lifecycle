# Multi-stage Dockerfile for User Service
# Stage 1: Builder - Compile the Go binary
FROM golang:1.21-alpine AS builder

# Create non-root user for the final image
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Copy go.mod and go.sum for dependency caching
# Note: go.sum must exist in the build context
COPY go.mod go.sum ./

# Set GOPROXY for reliable module resolution
ENV GOPROXY=https://proxy.golang.org,direct

# Download dependencies with caching
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG COMMIT=unknown

# Build static binary
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -trimpath \
    -o /app/bin/api \
    ./cmd/api

# Stage 2: Runtime - Minimal scratch image
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy user/group files for non-root user
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy the binary
COPY --from=builder /app/bin/api /api

# Run as non-root user
USER appuser:appuser

# Expose service port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/api", "healthcheck"]

# Metadata
LABEL org.opencontainers.image.source="https://github.com/Agustiniano/DevOps-Lifecycle"
LABEL org.opencontainers.image.description="User Service API"
LABEL org.opencontainers.image.licenses="MIT"

# Entrypoint
ENTRYPOINT ["/api"]
