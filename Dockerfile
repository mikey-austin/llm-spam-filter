FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go.mod and go.sum
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -o llm-spam-filter ./cmd/llm-spam-filter

# Create final image
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata sqlite

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Create directories
RUN mkdir -p /etc/llm-spam-filter /data && \
    chown -R appuser:appgroup /etc/llm-spam-filter /data

# Copy binary from builder
COPY --from=builder /app/llm-spam-filter /usr/local/bin/

# Copy config file
COPY configs/config.yaml /etc/llm-spam-filter/

# Set working directory
WORKDIR /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 10025

# Set data directory as volume
VOLUME ["/data"]

# Run the application
ENTRYPOINT ["llm-spam-filter"]
