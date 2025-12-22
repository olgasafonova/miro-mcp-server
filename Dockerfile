# Miro MCP Server Docker Image
# Multi-stage build for minimal final image

# Build stage
FROM golang:1.21-alpine AS builder

# Install git for fetching dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o miro-mcp-server .

# Final stage - minimal image
FROM alpine:3.19

# Add ca-certificates for HTTPS calls to Miro API
RUN apk add --no-cache ca-certificates

# Create non-root user for security
RUN adduser -D -u 1000 miro
USER miro

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/miro-mcp-server .

# MCP server runs on stdio by default
# For HTTP mode, use: -http :8080
EXPOSE 8080

ENTRYPOINT ["./miro-mcp-server"]
