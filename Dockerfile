# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for version info
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o miro-mcp-server .

# Runtime stage
FROM alpine:3.20

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/miro-mcp-server .

# Create non-root user
RUN adduser -D -u 1000 mcp
USER mcp

ENTRYPOINT ["./miro-mcp-server"]
