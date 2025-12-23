# Build stage
FROM golang:1.24-alpine AS builder

# Install git for fetching dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o miro-mcp-server .

# Final stage - minimal image
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/miro-mcp-server /miro-mcp-server

# Expose default HTTP port
EXPOSE 8080

# Set environment variables
ENV MIRO_TIMEOUT=30s

# Run the server in HTTP mode
ENTRYPOINT ["/miro-mcp-server"]
CMD ["-http", ":8080"]
