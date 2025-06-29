FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
  git \
  ca-certificates \
  tzdata \
  make

# Set working directory
WORKDIR /app

# Copy Go workspace files
COPY go.work go.work.sum ./

# Copy module files for better dependency caching
COPY shared/go.mod shared/go.sum ./shared/
COPY orchestrator/go.mod orchestrator/go.sum ./orchestrator/

# Download dependencies (cached layer)
RUN cd orchestrator && go mod download

# Copy source code
COPY shared/ ./shared/
COPY orchestrator/ ./orchestrator/

# Build the application
WORKDIR /app/orchestrator

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build the binary with version information
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-w -s \
  -X main.version=${VERSION} \
  -X main.commit=${COMMIT} \
  -X main.date=${BUILD_DATE} \
  -X main.builtBy=docker" \
  -a -installsuffix cgo \
  -o orchestrator .

# Verify the binary works
RUN ./orchestrator --version

# Production stage - minimal runtime image
FROM alpine:3.18

# Install runtime dependencies
RUN apk --no-cache add \
  ca-certificates \
  tzdata \
  curl \
  && adduser -D -s /bin/sh -u 65534 orchestrator

# Set timezone
ENV TZ=UTC

# Create necessary directories
RUN mkdir -p /app/data /app/logs /app/config /tmp/orchestrator && \
  chown -R orchestrator:orchestrator /app /tmp/orchestrator

# Copy the binary from builder stage
COPY --from=builder /app/orchestrator/orchestrator /app/orchestrator

# Copy any additional configuration files (if they exist)
COPY --from=builder --chown=orchestrator:orchestrator /app/orchestrator/deployments/examples/ /app/examples/

# Switch to non-root user
USER orchestrator

# Set working directory
WORKDIR /app

# Expose the application port
EXPOSE 8080

# Add labels for better container management
LABEL maintainer="platform-team@company.com" \
  version="${VERSION}" \
  description="Nestor Orchestrator - Infrastructure Component Catalog Manager" \
  org.opencontainers.image.title="nestor-orchestrator" \
  org.opencontainers.image.description="Nestor Orchestrator - Infrastructure Component Catalog Manager" \
  org.opencontainers.image.version="${VERSION}" \
  org.opencontainers.image.created="${BUILD_DATE}" \
  org.opencontainers.image.source="https://github.com/HatiCode/nestor" \
  org.opencontainers.image.licenses="Apache-2.0"

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# Default command
ENTRYPOINT ["/app/orchestrator"]
CMD ["serve"]
