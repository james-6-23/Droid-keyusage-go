.PHONY: help build run clean docker-build docker-up docker-down docker-logs test deps fmt lint

# Variables
BINARY_NAME=keyusage-server
DOCKER_IMAGE=keyusage:latest
GO=go
GOFLAGS=-v
MAIN_PATH=cmd/server/main.go

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

help: ## Display this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  ${GREEN}%-20s${NC} %s\n", $$1, $$2}'

deps: ## Download go module dependencies
	@echo "${YELLOW}Downloading dependencies...${NC}"
	$(GO) mod download
	$(GO) mod tidy

build: ## Build the application binary
	@echo "${YELLOW}Building application...${NC}"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="-w -s" -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "${GREEN}Build complete: $(BINARY_NAME)${NC}"

build-windows: ## Build for Windows
	@echo "${YELLOW}Building for Windows...${NC}"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="-w -s" -o $(BINARY_NAME).exe $(MAIN_PATH)
	@echo "${GREEN}Build complete: $(BINARY_NAME).exe${NC}"

build-mac: ## Build for macOS
	@echo "${YELLOW}Building for macOS...${NC}"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="-w -s" -o $(BINARY_NAME)-mac $(MAIN_PATH)
	@echo "${GREEN}Build complete: $(BINARY_NAME)-mac${NC}"

run: ## Run the application locally
	@echo "${YELLOW}Starting application...${NC}"
	$(GO) run $(MAIN_PATH)

run-dev: ## Run with development environment
	@echo "${YELLOW}Starting in development mode...${NC}"
	ENV=development $(GO) run $(MAIN_PATH)

clean: ## Clean build artifacts
	@echo "${YELLOW}Cleaning build artifacts...${NC}"
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe $(BINARY_NAME)-mac
	rm -rf logs/
	@echo "${GREEN}Clean complete${NC}"

test: ## Run tests
	@echo "${YELLOW}Running tests...${NC}"
	$(GO) test -v -race -cover ./...

test-coverage: ## Run tests with coverage report
	@echo "${YELLOW}Running tests with coverage...${NC}"
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "${GREEN}Coverage report generated: coverage.html${NC}"

benchmark: ## Run benchmarks
	@echo "${YELLOW}Running benchmarks...${NC}"
	$(GO) test -bench=. -benchmem ./...

fmt: ## Format code with gofmt
	@echo "${YELLOW}Formatting code...${NC}"
	$(GO) fmt ./...
	@echo "${GREEN}Format complete${NC}"

lint: ## Run linter
	@echo "${YELLOW}Running linter...${NC}"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "${RED}golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest${NC}"; \
	fi

vet: ## Run go vet
	@echo "${YELLOW}Running go vet...${NC}"
	$(GO) vet ./...

# Docker commands
docker-build: ## Build Docker image
	@echo "${YELLOW}Building Docker image...${NC}"
	docker build -f docker/Dockerfile -t $(DOCKER_IMAGE) .
	@echo "${GREEN}Docker image built: $(DOCKER_IMAGE)${NC}"

docker-up: ## Start services with docker-compose
	@echo "${YELLOW}Starting services...${NC}"
	docker-compose up -d
	@echo "${GREEN}Services started${NC}"

docker-down: ## Stop services
	@echo "${YELLOW}Stopping services...${NC}"
	docker-compose down
	@echo "${GREEN}Services stopped${NC}"

docker-logs: ## View container logs
	docker-compose logs -f app

docker-restart: docker-down docker-up ## Restart Docker services

docker-prod: ## Start production environment
	@echo "${YELLOW}Starting production environment...${NC}"
	docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
	@echo "${GREEN}Production environment started${NC}"

# Redis commands
redis-cli: ## Connect to Redis CLI
	docker-compose exec redis redis-cli

redis-flush: ## Flush Redis database
	@echo "${YELLOW}Flushing Redis database...${NC}"
	docker-compose exec redis redis-cli FLUSHDB
	@echo "${GREEN}Redis database flushed${NC}"

# Monitoring commands
monitor: ## Start monitoring stack (Prometheus + Grafana)
	@echo "${YELLOW}Starting monitoring stack...${NC}"
	docker-compose --profile monitoring up -d
	@echo "${GREEN}Monitoring available at: http://localhost:3000 (admin/admin)${NC}"

# Development helpers
dev-setup: deps ## Setup development environment
	@echo "${YELLOW}Setting up development environment...${NC}"
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "${GREEN}.env file created from .env.example${NC}"; \
	fi
	@mkdir -p logs
	@echo "${GREEN}Development environment ready${NC}"

watch: ## Watch for changes and rebuild
	@echo "${YELLOW}Watching for changes...${NC}"
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "${RED}Air not installed. Install with: go install github.com/cosmtrek/air@latest${NC}"; \
	fi

# Release commands
release: clean test build docker-build ## Create a release build
	@echo "${GREEN}Release build complete${NC}"

.DEFAULT_GOAL := help
