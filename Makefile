.PHONY: help build run test clean lint fmt deps

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	go build -o bin/auction ./cmd/auction

run: ## Run the application
	go run ./cmd/auction

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

coverage: test ## Run tests with coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts
	rm -rf bin/ dist/ coverage.out coverage.html

lint: ## Run linter
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...

deps: ## Download dependencies
	go mod download
	go mod tidy

vendor: ## Create vendor directory
	go mod vendor

install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

docker-build: ## Build Docker image
	docker build -t publika-auction:latest .

docker-run: ## Run Docker container
	docker run -p 8002:8002 --env-file .env publika-auction:latest

dev: ## Run in development mode with auto-reload (requires entr)
	find . -name "*.go" | entr -r make run
