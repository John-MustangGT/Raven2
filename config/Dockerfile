# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o raven ./cmd/raven

FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates curl

WORKDIR /app

# Copy the binary
COPY --from=builder /app/raven .

# Create data directory
RUN mkdir -p data plugins

# Expose ports
EXPOSE 8000 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8000/health || exit 1

CMD ["./raven", "-config", "config.yaml"]

