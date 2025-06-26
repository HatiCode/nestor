# Nestor Monorepo Makefile
# Unified build and development commands for all components

.PHONY: help setup build test lint clean dev docker release versions

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
PURPLE := \033[0;35m
CYAN := \033[0;36m
NC := \033[0m # No Color

# Go workspace and component directories
WORKSPACE_FILE := go.work
CLI_DIR := cli
ORCHESTRATOR_DIR := orchestrator
PROCESSOR_DIR := processor
SHARED_DIR := shared
TOOLS_DIR := tools

# Build directories
DIST_DIR := dist
COVERAGE_DIR := coverage
LOGS_DIR := logs

# Docker compose file
DOCKER_COMPOSE_FILE := docker-compose.yml

##@ Help

help: ## Display this help message
	@echo "$(CYAN)Nestor Development Commands$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make $(CYAN)<target>$(NC)\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  $(CYAN)%-15s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Setup & Environment

setup: ## Setup development environment (install tools, create directories, etc.)
	@echo "$(BLUE)[INFO]$(NC) Setting up Nestor development environment..."
	@./scripts/setup.sh
	@echo "$(GREEN)[SUCCESS]$(NC) Development environment ready!"

install-tools: ## Install development tools (golangci-lint, gosec, etc.)
	@echo "$(BLUE)[INFO]$(NC) Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@go install github.com/goreleaser/goreleaser@latest
	@go install github.com/aquasecurity/trivy@latest
	@echo "$(GREEN)[SUCCESS]$(NC) Development tools installed"

workspace-init: ## Initialize Go workspace
	@echo "$(BLUE)[INFO]$(NC) Initializing Go workspace..."
	@if [ ! -f $(WORKSPACE_FILE) ]; then \
		go work init $(CLI_DIR) $(ORCHESTRATOR_DIR) $(PROCESSOR_DIR) $(SHARED_DIR); \
		echo "$(GREEN)[SUCCESS]$(NC) Go workspace created"; \
	else \
		go work sync; \
		echo "$(GREEN)[SUCCESS]$(NC) Go workspace synchronized"; \
	fi

workspace-verify: ## Verify Go workspace is properly configured
	@echo "$(BLUE)[INFO]$(NC) Verifying Go workspace..."
	@go work verify
	@echo "$(GREEN)[SUCCESS]$(NC) Go workspace is valid"

create-dirs: ## Create necessary directories
	@echo "$(BLUE)[INFO]$(NC) Creating necessary directories..."
	@mkdir -p $(DIST_DIR) $(COVERAGE_DIR) $(LOGS_DIR)
	@echo "$(GREEN)[SUCCESS]$(NC) Directories created"

##@ Building

build: build-cli build-orchestrator build-processor ## Build all components
	@echo "$(GREEN)[SUCCESS]$(NC) All components built successfully"

build-cli: workspace-verify ## Build CLI component
	@echo "$(BLUE)[INFO]$(NC) Building CLI..."
	@cd $(CLI_DIR) && go build -o ../$(DIST_DIR)/nestor .
	@echo "$(GREEN)[SUCCESS]$(NC) CLI built: $(DIST_DIR)/nestor"

build-orchestrator: workspace-verify ## Build Orchestrator component
	@echo "$(BLUE)[INFO]$(NC) Building Orchestrator..."
	@cd $(ORCHESTRATOR_DIR) && go build -o ../$(DIST_DIR)/orchestrator .
	@echo "$(GREEN)[SUCCESS]$(NC) Orchestrator built: $(DIST_DIR)/orchestrator"

build-processor: workspace-verify ## Build Processor component
	@echo "$(BLUE)[INFO]$(NC) Building Processor..."
	@cd $(PROCESSOR_DIR) && go build -o ../$(DIST_DIR)/processor .
	@echo "$(GREEN)[SUCCESS]$(NC) Processor built: $(DIST_DIR)/processor"

build-release: ## Build release binaries for all platforms
	@echo "$(BLUE)[INFO]$(NC) Building release binaries..."
	@cd $(CLI_DIR) && goreleaser build --snapshot --clean
	@cd $(ORCHESTRATOR_DIR) && goreleaser build --snapshot --clean
	@cd $(PROCESSOR_DIR) && goreleaser build --snapshot --clean
	@echo "$(GREEN)[SUCCESS]$(NC) Release binaries built"

##@ Testing

test: test-shared test-cli test-orchestrator test-processor ## Run all tests
	@echo "$(GREEN)[SUCCESS]$(NC) All tests completed"

test-shared: workspace-verify ## Test shared libraries
	@echo "$(BLUE)[INFO]$(NC) Testing shared libraries..."
	@cd $(SHARED_DIR) && go test -v -race -coverprofile=../$(COVERAGE_DIR)/shared.out ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Shared library tests passed"

test-cli: workspace-verify ## Test CLI component
	@echo "$(BLUE)[INFO]$(NC) Testing CLI..."
	@cd $(CLI_DIR) && go test -v -race -coverprofile=../$(COVERAGE_DIR)/cli.out ./...
	@echo "$(GREEN)[SUCCESS]$(NC) CLI tests passed"

test-orchestrator: workspace-verify ## Test Orchestrator component
	@echo "$(BLUE)[INFO]$(NC) Testing Orchestrator..."
	@cd $(ORCHESTRATOR_DIR) && go test -v -race -coverprofile=../$(COVERAGE_DIR)/orchestrator.out ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Orchestrator tests passed"

test-processor: workspace-verify ## Test Processor component
	@echo "$(BLUE)[INFO]$(NC) Testing Processor..."
	@cd $(PROCESSOR_DIR) && go test -v -race -coverprofile=../$(COVERAGE_DIR)/processor.out ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Processor tests passed"

test-integration: workspace-verify ## Run integration tests
	@echo "$(BLUE)[INFO]$(NC) Running integration tests..."
	@cd test/integration && go test -v -race ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Integration tests passed"

test-watch: ## Run tests in watch mode (requires entr)
	@echo "$(BLUE)[INFO]$(NC) Starting test watch mode (Ctrl+C to stop)..."
	@find . -name "*.go" | entr -c make test

##@ Code Quality

lint: workspace-verify ## Run linters on all components
	@echo "$(BLUE)[INFO]$(NC) Running linters..."
	@golangci-lint run --config tools/lint/.golangci.yml
	@echo "$(GREEN)[SUCCESS]$(NC) Linting completed"

lint-fix: workspace-verify ## Run linters with auto-fix
	@echo "$(BLUE)[INFO]$(NC) Running linters with auto-fix..."
	@golangci-lint run --config tools/lint/.golangci.yml --fix
	@echo "$(GREEN)[SUCCESS]$(NC) Linting with auto-fix completed"

fmt: workspace-verify ## Format all Go code
	@echo "$(BLUE)[INFO]$(NC) Formatting Go code..."
	@go fmt ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Code formatting completed"

vet: workspace-verify ## Run go vet on all components
	@echo "$(BLUE)[INFO]$(NC) Running go vet..."
	@go vet ./...
	@echo "$(GREEN)[SUCCESS]$(NC) go vet completed"

security: workspace-verify ## Run security scans
	@echo "$(BLUE)[INFO]$(NC) Running security scans..."
	@gosec -fmt=json -out=$(LOGS_DIR)/gosec.json ./...
	@trivy fs --format json --output $(LOGS_DIR)/trivy.json .
	@echo "$(GREEN)[SUCCESS]$(NC) Security scans completed"
	@echo "$(CYAN)[INFO]$(NC) Results saved to $(LOGS_DIR)/"

##@ Coverage

coverage: test ## Generate coverage report for all components
	@echo "$(BLUE)[INFO]$(NC) Generating coverage reports..."
	@mkdir -p $(COVERAGE_DIR)
	@go tool cover -html=$(COVERAGE_DIR)/cli.out -o $(COVERAGE_DIR)/cli.html
	@go tool cover -html=$(COVERAGE_DIR)/orchestrator.out -o $(COVERAGE_DIR)/orchestrator.html
	@go tool cover -html=$(COVERAGE_DIR)/processor.out -o $(COVERAGE_DIR)/processor.html
	@go tool cover -html=$(COVERAGE_DIR)/shared.out -o $(COVERAGE_DIR)/shared.html
	@echo "$(GREEN)[SUCCESS]$(NC) Coverage reports generated in $(COVERAGE_DIR)/"

coverage-summary: test ## Show coverage summary
	@echo "$(BLUE)[INFO]$(NC) Coverage Summary:"
	@echo "$(CYAN)CLI:$(NC)"
	@cd $(CLI_DIR) && go tool cover -func=../$(COVERAGE_DIR)/cli.out | tail -1
	@echo "$(CYAN)Orchestrator:$(NC)"
	@cd $(ORCHESTRATOR_DIR) && go tool cover -func=../$(COVERAGE_DIR)/orchestrator.out | tail -1
	@echo "$(CYAN)Processor:$(NC)"
	@cd $(PROCESSOR_DIR) && go tool cover -func=../$(COVERAGE_DIR)/processor.out | tail -1
	@echo "$(CYAN)Shared:$(NC)"
	@cd $(SHARED_DIR) && go tool cover -func=../$(COVERAGE_DIR)/shared.out | tail -1

##@ Development

dev-orchestrator: build-orchestrator ## Run Orchestrator in development mode
	@echo "$(BLUE)[INFO]$(NC) Starting Orchestrator in development mode..."
	@cd $(ORCHESTRATOR_DIR) && ./$(DIST_DIR)/orchestrator serve --config configs/development.yaml

dev-processor: build-processor ## Run Processor in development mode
	@echo "$(BLUE)[INFO]$(NC) Starting Processor in development mode..."
	@cd $(PROCESSOR_DIR) && ./$(DIST_DIR)/processor --config configs/development.yaml

dev-cli: build-cli ## Test CLI in development mode
	@echo "$(BLUE)[INFO]$(NC) CLI ready for testing:"
	@echo "$(CYAN)Usage:$(NC) ./$(DIST_DIR)/nestor [command]"
	@./$(DIST_DIR)/nestor --help

##@ Docker

docker-build: ## Build Docker images for all components
	@echo "$(BLUE)[INFO]$(NC) Building Docker images..."
	@docker build -t nestor/orchestrator:dev -f $(ORCHESTRATOR_DIR)/Dockerfile .
	@docker build -t nestor/processor:dev -f $(PROCESSOR_DIR)/Dockerfile .
	@echo "$(GREEN)[SUCCESS]$(NC) Docker images built"

docker-up: ## Start development environment with Docker Compose
	@echo "$(BLUE)[INFO]$(NC) Starting Docker Compose environment..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d
	@echo "$(GREEN)[SUCCESS]$(NC) Development environment started"
	@echo "$(CYAN)[INFO]$(NC) Services available:"
	@echo "  - Orchestrator: http://localhost:8080"
	@echo "  - DynamoDB Local: http://localhost:8000"
	@echo "  - Redis: localhost:6379"

docker-down: ## Stop Docker Compose environment
	@echo "$(BLUE)[INFO]$(NC) Stopping Docker Compose environment..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down
	@echo "$(GREEN)[SUCCESS]$(NC) Development environment stopped"

docker-logs: ## Show Docker Compose logs
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

docker-clean: ## Clean Docker images and containers
	@echo "$(BLUE)[INFO]$(NC) Cleaning Docker resources..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down -v --remove-orphans
	@docker image prune -f
	@docker container prune -f
	@echo "$(GREEN)[SUCCESS]$(NC) Docker cleanup completed"

##@ Dependencies

deps: workspace-verify ## Update dependencies for all components
	@echo "$(BLUE)[INFO]$(NC) Updating dependencies..."
	@go work sync
	@cd $(CLI_DIR) && go mod tidy
	@cd $(ORCHESTRATOR_DIR) && go mod tidy
	@cd $(PROCESSOR_DIR) && go mod tidy
	@cd $(SHARED_DIR) && go mod tidy
	@echo "$(GREEN)[SUCCESS]$(NC) Dependencies updated"

deps-upgrade: workspace-verify ## Upgrade dependencies for all components
	@echo "$(BLUE)[INFO]$(NC) Upgrading dependencies..."
	@cd $(CLI_DIR) && go get -u ./... && go mod tidy
	@cd $(ORCHESTRATOR_DIR) && go get -u ./... && go mod tidy
	@cd $(PROCESSOR_DIR) && go get -u ./... && go mod tidy
	@cd $(SHARED_DIR) && go get -u ./... && go mod tidy
	@go work sync
	@echo "$(GREEN)[SUCCESS]$(NC) Dependencies upgraded"

deps-verify: workspace-verify ## Verify dependencies are consistent
	@echo "$(BLUE)[INFO]$(NC) Verifying dependencies..."
	@go mod verify
	@echo "$(GREEN)[SUCCESS]$(NC) Dependencies verified"

##@ Releases

release-cli: ## Tag and release CLI component
	@echo "$(BLUE)[INFO]$(NC) Releasing CLI component..."
	@./scripts/release.sh cli

release-orchestrator: ## Tag and release Orchestrator component
	@echo "$(BLUE)[INFO]$(NC) Releasing Orchestrator component..."
	@./scripts/release.sh orchestrator

release-processor: ## Tag and release Processor component
	@echo "$(BLUE)[INFO]$(NC) Releasing Processor component..."
	@./scripts/release.sh processor

release-shared: ## Tag and release Shared libraries
	@echo "$(BLUE)[INFO]$(NC) Releasing Shared libraries..."
	@./scripts/release.sh shared

versions: ## Show current version of all components
	@echo "$(BLUE)[INFO]$(NC) Component Versions:"
	@echo "$(CYAN)CLI:$(NC)          $$(git tag -l 'cli/v*' | sort -V | tail -1 | sed 's/cli\///' || echo 'v0.0.0')"
	@echo "$(CYAN)Orchestrator:$(NC) $$(git tag -l 'orchestrator/v*' | sort -V | tail -1 | sed 's/orchestrator\///' || echo 'v0.0.0')"
	@echo "$(CYAN)Processor:$(NC)    $$(git tag -l 'processor/v*' | sort -V | tail -1 | sed 's/processor\///' || echo 'v0.0.0')"
	@echo "$(CYAN)Shared:$(NC)       $$(git tag -l 'shared/v*' | sort -V | tail -1 | sed 's/shared\///' || echo 'v0.0.0')"

release-status: ## Check status of recent releases
	@echo "$(BLUE)[INFO]$(NC) Recent Release Status:"
	@./scripts/release-status.sh

##@ Code Generation

generate: workspace-verify ## Run code generation
	@echo "$(BLUE)[INFO]$(NC) Running code generation..."
	@go generate ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Code generation completed"

mocks: workspace-verify ## Generate mocks for testing
	@echo "$(BLUE)[INFO]$(NC) Generating mocks..."
	@go run $(TOOLS_DIR)/generate/main.go mocks
	@echo "$(GREEN)[SUCCESS]$(NC) Mocks generated"

##@ Documentation

docs-serve: ## Serve documentation locally
	@echo "$(BLUE)[INFO]$(NC) Starting documentation server..."
	@if [ -d docs ] && [ -f docs/package.json ]; then \
		cd docs && npm run dev; \
	else \
		echo "$(RED)[ERROR]$(NC) Documentation dependencies not found. Run 'make setup' first."; \
	fi

docs-build: ## Build documentation
	@echo "$(BLUE)[INFO]$(NC) Building documentation..."
	@if [ -d docs ] && [ -f docs/package.json ]; then \
		cd docs && npm run build; \
		echo "$(GREEN)[SUCCESS]$(NC) Documentation built"; \
	else \
		echo "$(RED)[ERROR]$(NC) Documentation dependencies not found. Run 'make setup' first."; \
	fi

docs-lint: ## Lint documentation
	@echo "$(BLUE)[INFO]$(NC) Linting documentation..."
	@markdownlint docs/ README.md CONTRIBUTING.md
	@echo "$(GREEN)[SUCCESS]$(NC) Documentation linting completed"

##@ Git Hooks

install-hooks: ## Install Git hooks for development
	@echo "$(BLUE)[INFO]$(NC) Installing Git hooks..."
	@./scripts/install-hooks.sh
	@echo "$(GREEN)[SUCCESS]$(NC) Git hooks installed"

##@ Cleanup

clean: ## Clean build artifacts and cache
	@echo "$(BLUE)[INFO]$(NC) Cleaning build artifacts..."
	@rm -rf $(DIST_DIR)
	@rm -rf $(COVERAGE_DIR)
	@rm -rf $(LOGS_DIR)
	@go clean -cache
	@go clean -modcache
	@echo "$(GREEN)[SUCCESS]$(NC) Cleanup completed"

clean-docker: docker-clean ## Clean Docker resources

clean-all: clean clean-docker ## Clean everything (build artifacts, cache, Docker)
	@echo "$(GREEN)[SUCCESS]$(NC) Full cleanup completed"

##@ Checks & Validation

check: lint test security ## Run all quality checks (lint, test, security)
	@echo "$(GREEN)[SUCCESS]$(NC) All quality checks passed!"

validate: workspace-verify deps-verify fmt vet ## Run all validation checks
	@echo "$(GREEN)[SUCCESS]$(NC) All validation checks passed!"

ci: setup check build ## Run CI checks locally (same as GitHub Actions)
	@echo "$(GREEN)[SUCCESS]$(NC) CI checks completed successfully!"

pre-commit: fmt lint test ## Run pre-commit checks
	@echo "$(GREEN)[SUCCESS]$(NC) Pre-commit checks passed!"

##@ Utilities

info: ## Show development environment information
	@echo "$(CYAN)Nestor Development Environment Info$(NC)"
	@echo "$(YELLOW)Go Version:$(NC)        $$(go version)"
	@echo "$(YELLOW)Go Workspace:$(NC)      $$(if [ -f $(WORKSPACE_FILE) ]; then echo "✅ Configured"; else echo "❌ Not found"; fi)"
	@echo "$(YELLOW)Docker:$(NC)            $$(if command -v docker >/dev/null 2>&1; then echo "✅ Available"; else echo "❌ Not installed"; fi)"
	@echo "$(YELLOW)Node.js:$(NC)           $$(if command -v node >/dev/null 2>&1; then echo "✅ $$(node --version)"; else echo "❌ Not installed"; fi)"
	@echo "$(YELLOW)Build Tools:$(NC)"
	@echo "  golangci-lint:        $$(if command -v golangci-lint >/dev/null 2>&1; then echo "✅ Available"; else echo "❌ Not installed"; fi)"
	@echo "  gosec:                $$(if command -v gosec >/dev/null 2>&1; then echo "✅ Available"; else echo "❌ Not installed"; fi)"
	@echo "  goreleaser:           $$(if command -v goreleaser >/dev/null 2>&1; then echo "✅ Available"; else echo "❌ Not installed"; fi)"

watch: ## Watch for file changes and run tests
	@echo "$(BLUE)[INFO]$(NC) Watching for file changes..."
	@if command -v entr >/dev/null 2>&1; then \
		find . -name "*.go" -not -path "./vendor/*" | entr -c make test; \
	else \
		echo "$(RED)[ERROR]$(NC) entr not installed. Install with: brew install entr (macOS) or apt install entr (Ubuntu)"; \
	fi

# Include component-specific Makefiles if they exist
-include $(CLI_DIR)/Makefile.local
-include $(ORCHESTRATOR_DIR)/Makefile.local
-include $(PROCESSOR_DIR)/Makefile.local
