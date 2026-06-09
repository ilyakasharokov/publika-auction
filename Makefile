.PHONY: help build run test clean lint fmt deps

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	go build -o bin/auction ./cmd

run: ## Run the application
	go run ./cmd

test: ## Run unit tests (no infrastructure required)
	go test -v -race -coverprofile=coverage.out ./internal/...

test-integration: ## Run integration tests (requires running MongoDB + Redis)
	go test -v -race -tags=integration -count=1 ./test/...

test-all: test test-integration ## Run all tests

bench: ## Run Go benchmarks (unit, no infrastructure)
	go test -bench=. -benchmem -benchtime=5s ./internal/service/bid/...

load: ## Run load tests against local infra (requires MongoDB + Redis)
	go test -v -tags=load -timeout=60s ./test/...

k6: ## Run k6 HTTP load test (requires app running + k6 installed)
	k6 run -e AUCTION_SLUG=$(SLUG) -e LOT_NUM=$(LOT) test/k6/admin_panel.js

coverage: test ## Open HTML coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

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
