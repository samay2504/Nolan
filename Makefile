.PHONY: secrets proto build up down clean test-go test-rust test lint logs help

secrets: ## Generate secrets and .env file
	@if [ -f "scripts/gen-secrets.sh" ]; then bash scripts/gen-secrets.sh; else powershell -ExecutionPolicy Bypass -File scripts/gen-secrets.ps1; fi

proto: ## Generate protobuf code
	@cd proto && buf generate

build: ## Build docker images
	@docker compose build

up: ## Start all services in background
	@docker compose up -d

down: ## Stop all services
	@docker compose down

clean: ## Stop all services and remove volumes
	@docker compose down -v

test-go: ## Run Go tests
	@cd control-plane && go test ./...

test-rust: ## Run Rust tests
	@cd worker && cargo test

test: test-go test-rust ## Run all tests

lint: ## Lint proto and Go code
	@cd proto && buf lint
	@cd control-plane && go vet ./...

logs: ## Tail docker logs
	@docker compose logs -f

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
