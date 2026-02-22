# Miro MCP Server Makefile

# Variables
BINARY_NAME=miro-mcp-server
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-w -s -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Default target
.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the binary
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

.PHONY: build-linux
build-linux: ## Build for Linux (amd64)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .

.PHONY: build-all
build-all: ## Build for all platforms
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .

.PHONY: run
run: build ## Build and run the server (stdio mode)
	./$(BINARY_NAME)

.PHONY: run-http
run-http: build ## Build and run the server (HTTP mode on :8080)
	./$(BINARY_NAME) -http :8080

.PHONY: test
test: ## Run tests
	$(GOTEST) -v ./...

.PHONY: test-cover
test-cover: ## Run tests with coverage
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: test-race
test-race: ## Run tests with race detector
	$(GOTEST) -v -race ./...

.PHONY: lint
lint: ## Run linter
	golangci-lint run

.PHONY: vet
vet: ## Run go vet
	$(GOVET) ./...

.PHONY: fmt
fmt: ## Format code
	$(GOFMT) -w -s .

.PHONY: fmt-check
fmt-check: ## Check code formatting
	@test -z "$$($(GOFMT) -l .)" || (echo "Code not formatted. Run 'make fmt'" && exit 1)

.PHONY: check
check: fmt-check vet lint test ## Run all quality checks

.PHONY: tidy
tidy: ## Tidy go modules
	$(GOMOD) tidy

.PHONY: deps
deps: ## Download dependencies
	$(GOMOD) download

.PHONY: token-badge
token-badge: ## Generate MCP token count badge
	go run ./cmd/token-count/

.PHONY: clean
clean: ## Clean build artifacts
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -rf dist/
	rm -f coverage.out coverage.html

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t $(BINARY_NAME):$(VERSION) -t $(BINARY_NAME):latest .

.PHONY: docker-run
docker-run: ## Run Docker container (requires MIRO_ACCESS_TOKEN env var)
	docker run --rm -e MIRO_ACCESS_TOKEN -p 8080:8080 $(BINARY_NAME):latest

.PHONY: docker-compose-up
docker-compose-up: ## Start with docker-compose
	docker-compose up -d

.PHONY: docker-compose-down
docker-compose-down: ## Stop docker-compose
	docker-compose down

.PHONY: install
install: build ## Install binary to GOPATH/bin
	cp $(BINARY_NAME) $(GOPATH)/bin/

.PHONY: all
all: clean deps fmt vet lint test build ## Run all checks and build
