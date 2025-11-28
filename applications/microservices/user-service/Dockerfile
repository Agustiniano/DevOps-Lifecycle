#Test

# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Security: Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go.mod first for caching; generate go.sum inside the build
# (CI runner may not have `go` locally; let the builder download modules)
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG COMMIT=unknown

# Build with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -trimpath \
    -o /app/bin/api \
    ./cmd/api

# Stage 2: Runtime
FROM scratch

# Copy SSL certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy user information
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy binary
COPY --from=builder /app/bin/api /api

# Use non-root user
USER appuser:appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/api", "healthcheck"]

# Metadata labels
LABEL org.opencontainers.image.source="https://github.com/acme/devops-lifecycle"
LABEL org.opencontainers.image.description="User Service API"
LABEL org.opencontainers.image.licenses="MIT"

# Run
ENTRYPOINT ["/api"]
