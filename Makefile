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

# Go version management - centralized source of truth
GO_VERSION := 1.24.4
REQUIRED_GO_VERSION := 1.24

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

# Build metadata
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")

# Default LDFLAGS for all builds
BASE_LDFLAGS := -s -w
BASE_LDFLAGS += -X main.version=$(GIT_VERSION)
BASE_LDFLAGS += -X main.commit=$(GIT_COMMIT)
BASE_LDFLAGS += -X main.date=$(BUILD_DATE)
BASE_LDFLAGS += -X main.builtBy=makefile

##@ Help

help: ## Display this help message
	@echo "$(CYAN)Nestor Development Commands$(NC)"
	@echo ""
	@echo "$(YELLOW)Go Version:$(NC) $(GO_VERSION)"
	@echo "$(YELLOW)Git Commit:$(NC) $(GIT_COMMIT)"
	@echo "$(YELLOW)Build Date:$(NC) $(BUILD_DATE)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make $(CYAN)<target>$(NC)\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  $(CYAN)%-15s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Validation & Environment

check-go-version: ## Check if Go version meets requirements
	@echo "$(BLUE)[INFO]$(NC) Checking Go version..."
	@go_version=$$(go version | grep -oE 'go[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1); \
	if [ -z "$$go_version" ]; then \
		echo "$(RED)[ERROR]$(NC) Go not found. Please install Go $(REQUIRED_GO_VERSION)+"; \
		exit 1; \
	fi; \
	required_version="go$(REQUIRED_GO_VERSION)"; \
	if printf '%s\n%s\n' "$$required_version" "$$go_version" | sort -V -C; then \
		echo "$(GREEN)[SUCCESS]$(NC) Go version: $$go_version"; \
	else \
		echo "$(RED)[ERROR]$(NC) Go version $$go_version is too old. Required: $(REQUIRED_GO_VERSION)+"; \
		exit 1; \
	fi

check-workspace: check-go-version ## Verify Go workspace is properly configured
	@echo "$(BLUE)[INFO]$(NC) Checking Go workspace..."
	@if [ ! -f $(WORKSPACE_FILE) ]; then \
		echo "$(RED)[ERROR]$(NC) $(WORKSPACE_FILE) not found. Run 'make workspace-init'"; \
		exit 1; \
	fi
	@go work verify
	@echo "$(GREEN)[SUCCESS]$(NC) Go workspace is valid"

##@ Setup & Environment

setup: install-tools workspace-init create-dirs ## Setup complete development environment
	@echo "$(GREEN)[SUCCESS]$(NC) Development environment ready!"
	@echo "$(CYAN)[INFO]$(NC) Run 'make check' to verify everything is working"

install-tools: check-go-version ## Install development tools
	@echo "$(BLUE)[INFO]$(NC) Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@go install github.com/goreleaser/goreleaser@latest
	@go install github.com/aquasecurity/trivy@latest
	@go install github.com/air-verse/air@latest
	@echo "$(GREEN)[SUCCESS]$(NC) Development tools installed"

workspace-init: check-go-version ## Initialize or update Go workspace
	@echo "$(BLUE)[INFO]$(NC) Setting up Go workspace..."
	@if [ ! -f $(WORKSPACE_FILE) ]; then \
		echo "$(BLUE)[INFO]$(NC) Creating Go workspace..."; \
		echo "go $(GO_VERSION)" > $(WORKSPACE_FILE); \
		echo "" >> $(WORKSPACE_FILE); \
		echo "use (" >> $(WORKSPACE_FILE); \
		echo "	./$(CLI_DIR)" >> $(WORKSPACE_FILE); \
		echo "	./$(ORCHESTRATOR_DIR)" >> $(WORKSPACE_FILE); \
		echo "	./$(PROCESSOR_DIR)" >> $(WORKSPACE_FILE); \
		echo "	./$(SHARED_DIR)" >> $(WORKSPACE_FILE); \
		echo ")" >> $(WORKSPACE_FILE); \
		echo "$(GREEN)[SUCCESS]$(NC) Go workspace created"; \
	fi
	@go work sync
	@echo "$(GREEN)[SUCCESS]$(NC) Go workspace synchronized"

create-dirs: ## Create necessary directories
	@echo "$(BLUE)[INFO]$(NC) Creating necessary directories..."
	@mkdir -p $(DIST_DIR) $(COVERAGE_DIR) $(LOGS_DIR)
	@mkdir -p $(TOOLS_DIR)/{lint,generate}
	@mkdir -p test/integration
	@mkdir -p deployments/{helm,docker,k8s}
	@echo "$(GREEN)[SUCCESS]$(NC) Directories created"

##@ Building

build: build-cli build-orchestrator build-processor ## Build all components
	@echo "$(GREEN)[SUCCESS]$(NC) All components built successfully"

build-cli: check-workspace ## Build CLI component
	@echo "$(BLUE)[INFO]$(NC) Building CLI..."
	@cd $(CLI_DIR) && go build -ldflags="$(BASE_LDFLAGS)" -o ../$(DIST_DIR)/nestor .
	@echo "$(GREEN)[SUCCESS]$(NC) CLI built: $(DIST_DIR)/nestor"

build-orchestrator: check-workspace ## Build Orchestrator component
	@echo "$(BLUE)[INFO]$(NC) Building Orchestrator..."
	@cd $(ORCHESTRATOR_DIR) && go build -ldflags="$(BASE_LDFLAGS)" -o ../$(DIST_DIR)/orchestrator .
	@echo "$(GREEN)[SUCCESS]$(NC) Orchestrator built: $(DIST_DIR)/orchestrator"

build-processor: check-workspace ## Build Processor component
	@echo "$(BLUE)[INFO]$(NC) Building Processor..."
	@cd $(PROCESSOR_DIR) && go build -ldflags="$(BASE_LDFLAGS)" -o ../$(DIST_DIR)/processor .
	@echo "$(GREEN)[SUCCESS]$(NC) Processor built: $(DIST_DIR)/processor"

build-all-platforms: check-workspace ## Build release binaries for all platforms
	@echo "$(BLUE)[INFO]$(NC) Building for all platforms..."
	@for component in cli orchestrator processor; do \
		echo "$(BLUE)[INFO]$(NC) Building $component for multiple platforms..."; \
		cd $component && \
		for os in linux darwin windows; do \
			for arch in amd64 arm64; do \
				if [ "$os" = "windows" ] && [ "$arch" = "arm64" ]; then continue; fi; \
				ext=""; if [ "$os" = "windows" ]; then ext=".exe"; fi; \
				echo "  Building $component for $os/$arch..."; \
				GOOS=$os GOARCH=$arch CGO_ENABLED=0 \
					go build -ldflags="$(BASE_LDFLAGS)" \
					-o "../$(DIST_DIR)/$component-$os-$arch$ext" . || exit 1; \
			done; \
		done; \
		cd ..; \
	done
	@echo "$(GREEN)[SUCCESS]$(NC) Multi-platform builds completed"

##@ Testing

test: test-workspace test-all ## Run all tests with workspace validation
	@echo "$(GREEN)[SUCCESS]$(NC) All tests completed successfully"

test-workspace: check-workspace ## Validate workspace before running tests
	@echo "$(BLUE)[INFO]$(NC) Validating workspace for testing..."
	@go work sync
	@go mod download

test-all: test-shared test-cli test-orchestrator test-processor ## Run tests for all components
	@echo "$(BLUE)[INFO]$(NC) Running tests for all components..."

test-shared: check-workspace ## Test shared libraries
	@echo "$(BLUE)[INFO]$(NC) Testing shared libraries..."
	@cd $(SHARED_DIR) && go test -v -race -coverprofile=../$(COVERAGE_DIR)/shared.out -covermode=atomic ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Shared library tests passed"

test-cli: check-workspace ## Test CLI component
	@echo "$(BLUE)[INFO]$(NC) Testing CLI..."
	@cd $(CLI_DIR) && go test -v -race -coverprofile=../$(COVERAGE_DIR)/cli.out -covermode=atomic ./...
	@echo "$(GREEN)[SUCCESS]$(NC) CLI tests passed"

test-orchestrator: check-workspace ## Test Orchestrator component
	@echo "$(BLUE)[INFO]$(NC) Testing Orchestrator..."
	@cd $(ORCHESTRATOR_DIR) && go test -v -race -coverprofile=../$(COVERAGE_DIR)/orchestrator.out -covermode=atomic ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Orchestrator tests passed"

test-processor: check-workspace ## Test Processor component
	@echo "$(BLUE)[INFO]$(NC) Testing Processor..."
	@cd $(PROCESSOR_DIR) && go test -v -race -coverprofile=../$(COVERAGE_DIR)/processor.out -covermode=atomic ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Processor tests passed"

test-integration: check-workspace docker-up-test ## Run integration tests with dependencies
	@echo "$(BLUE)[INFO]$(NC) Running integration tests..."
	@sleep 5  # Give services time to start
	@if [ ! -d test/integration ]; then mkdir -p test/integration; fi
	@if [ ! -f test/integration/go.mod ]; then \
		cd test/integration && go mod init nestor-integration-tests; \
	fi
	@cd test/integration && go test -v -race ./... || (make docker-down-test && exit 1)
	@make docker-down-test
	@echo "$(GREEN)[SUCCESS]$(NC) Integration tests passed"

test-watch: check-workspace ## Run tests in watch mode (requires entr)
	@echo "$(BLUE)[INFO]$(NC) Starting test watch mode (Ctrl+C to stop)..."
	@if command -v entr >/dev/null 2>&1; then \
		find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | entr -c make test-all; \
	else \
		echo "$(RED)[ERROR]$(NC) entr not installed. Install with: brew install entr (macOS) or apt install entr (Ubuntu)"; \
		exit 1; \
	fi

##@ Code Quality

lint: check-workspace ## Run linters on all components
	@echo "$(BLUE)[INFO]$(NC) Running linters..."
	@if [ ! -f $(TOOLS_DIR)/lint/.golangci.yml ]; then \
		echo "$(YELLOW)[WARNING]$(NC) golangci-lint config not found, creating default..."; \
		mkdir -p $(TOOLS_DIR)/lint; \
		make create-lint-config; \
	fi
	@golangci-lint run --config $(TOOLS_DIR)/lint/.golangci.yml
	@echo "$(GREEN)[SUCCESS]$(NC) Linting completed"

lint-fix: check-workspace ## Run linters with auto-fix
	@echo "$(BLUE)[INFO]$(NC) Running linters with auto-fix..."
	@golangci-lint run --config $(TOOLS_DIR)/lint/.golangci.yml --fix
	@echo "$(GREEN)[SUCCESS]$(NC) Linting with auto-fix completed"

fmt: check-workspace ## Format all Go code
	@echo "$(BLUE)[INFO]$(NC) Formatting Go code..."
	@go fmt ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Code formatting completed"

vet: check-workspace ## Run go vet on all components
	@echo "$(BLUE)[INFO]$(NC) Running go vet..."
	@go vet ./...
	@echo "$(GREEN)[SUCCESS]$(NC) go vet completed"

security: check-workspace ## Run security scans
	@echo "$(BLUE)[INFO]$(NC) Running security scans..."
	@mkdir -p $(LOGS_DIR)
	@echo "$(BLUE)[INFO]$(NC) Running gosec..."
	@gosec -fmt=json -out=$(LOGS_DIR)/gosec.json ./... || echo "$(YELLOW)[WARNING]$(NC) gosec found issues"
	@echo "$(BLUE)[INFO]$(NC) Running trivy..."
	@trivy fs --format json --output $(LOGS_DIR)/trivy.json . || echo "$(YELLOW)[WARNING]$(NC) trivy found issues"
	@echo "$(GREEN)[SUCCESS]$(NC) Security scans completed"
	@echo "$(CYAN)[INFO]$(NC) Results saved to $(LOGS_DIR)/"

create-lint-config: ## Create default golangci-lint configuration
	@mkdir -p $(TOOLS_DIR)/lint
	@cat > $(TOOLS_DIR)/lint/.golangci.yml << 'EOF'
run:
  timeout: 5m
  modules-download-mode: readonly
  allow-parallel-runners: true

output:
  format: colored-line-number
  print-issued-lines: true
  print-linter-name: true

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - typecheck
    - unused
    - gci
    - gofmt
    - goimports
    - gosec
    - misspell
    - revive
    - unparam
    - unconvert
    - gocritic
    - gocyclo
    - dupl
    - goconst
    - gofumpt
    - goprintffuncname
    - nolintlint

linters-settings:
  gci:
    sections:
      - standard
      - default
      - prefix(github.com/HatiCode/nestor)
  revive:
    min-confidence: 0.8
  gocritic:
    enabled-tags:
      - diagnostic
      - experimental
      - opinionated
      - performance
      - style
  gocyclo:
    min-complexity: 15
  goconst:
    min-len: 2
    min-occurrences: 3

issues:
  exclude-use-default: false
  exclude-rules:
    - path: _test\.go
      linters:
        - gosec
        - dupl
        - goconst
    - path: main\.go
      linters:
        - gocyclo
    - text: "weak cryptographic primitive"
      linters:
        - gosec
    - text: "should not use dot imports"
      linters:
        - revive
  max-issues-per-linter: 0
  max-same-issues: 0
EOF

##@ Coverage

coverage: test ## Generate coverage report for all components
	@echo "$(BLUE)[INFO]$(NC) Generating coverage reports..."
	@mkdir -p $(COVERAGE_DIR)
	@for component in cli orchestrator processor shared; do \
		if [ -f $(COVERAGE_DIR)/$component.out ]; then \
			echo "$(BLUE)[INFO]$(NC) Generating HTML coverage for $component..."; \
			go tool cover -html=$(COVERAGE_DIR)/$component.out -o $(COVERAGE_DIR)/$component.html; \
		fi; \
	done
	@echo "$(GREEN)[SUCCESS]$(NC) Coverage reports generated in $(COVERAGE_DIR)/"

coverage-summary: test ## Show coverage summary for all components
	@echo "$(BLUE)[INFO]$(NC) Coverage Summary:"
	@for component in cli orchestrator processor shared; do \
		if [ -f $(COVERAGE_DIR)/$component.out ]; then \
			echo "$(CYAN)$component:$(NC)"; \
			go tool cover -func=$(COVERAGE_DIR)/$component.out | tail -1; \
		else \
			echo "$(YELLOW)$component: No coverage data$(NC)"; \
		fi; \
	done

coverage-total: test ## Calculate total coverage across all components
	@echo "$(BLUE)[INFO]$(NC) Calculating total coverage..."
	@if command -v gocovmerge >/dev/null 2>&1; then \
		gocovmerge $(COVERAGE_DIR)/*.out > $(COVERAGE_DIR)/total.out; \
		echo "$(CYAN)Total Coverage:$(NC)"; \
		go tool cover -func=$(COVERAGE_DIR)/total.out | tail -1; \
	else \
		echo "$(YELLOW)[WARNING]$(NC) gocovmerge not installed. Install with: go install github.com/wadey/gocovmerge@latest"; \
	fi

##@ Development

dev-orchestrator: build-orchestrator ## Run Orchestrator in development mode
	@echo "$(BLUE)[INFO]$(NC) Starting Orchestrator in development mode..."
	@if [ ! -f $(ORCHESTRATOR_DIR)/configs/development.yaml ]; then \
		echo "$(YELLOW)[WARNING]$(NC) Development config not found, creating basic config..."; \
		mkdir -p $(ORCHESTRATOR_DIR)/configs; \
		make create-dev-config; \
	fi
	@cd $(ORCHESTRATOR_DIR) && ../$(DIST_DIR)/orchestrator serve --config configs/development.yaml

dev-processor: build-processor ## Run Processor in development mode
	@echo "$(BLUE)[INFO]$(NC) Starting Processor in development mode..."
	@if [ ! -f $(PROCESSOR_DIR)/configs/development.yaml ]; then \
		echo "$(YELLOW)[WARNING]$(NC) Development config not found, creating basic config..."; \
		mkdir -p $(PROCESSOR_DIR)/configs; \
		make create-dev-config; \
	fi
	@cd $(PROCESSOR_DIR) && ../$(DIST_DIR)/processor --config configs/development.yaml

dev-cli: build-cli ## Test CLI in development mode
	@echo "$(BLUE)[INFO]$(NC) CLI ready for testing:"
	@echo "$(CYAN)Usage:$(NC) ./$(DIST_DIR)/nestor [command]"
	@./$(DIST_DIR)/nestor --help || echo "$(YELLOW)[INFO]$(NC) CLI help not implemented yet"

dev-watch: ## Run development with auto-reload (requires air)
	@echo "$(BLUE)[INFO]$(NC) Starting development with auto-reload..."
	@if command -v air >/dev/null 2>&1; then \
		air -c .air.toml; \
	else \
		echo "$(RED)[ERROR]$(NC) air not installed. Install with: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi

create-dev-config: ## Create basic development configuration files
	@echo "$(BLUE)[INFO]$(NC) Creating development configuration..."
	@for component in orchestrator processor; do \
		mkdir -p $component/configs; \
		cat > $component/configs/development.yaml << 'EOF'
# Development configuration for $component
log:
  level: debug
  format: console

server:
  host: localhost
  port: 8080

database:
  endpoint: http://localhost:8000

redis:
  url: redis://localhost:6379

development:
  hot_reload: true
  debug: true
EOF
	done

##@ Docker

docker-build: ## Build Docker images for all components
	@echo "$(BLUE)[INFO]$(NC) Building Docker images..."
	@for component in orchestrator processor; do \
		if [ -f $component/Dockerfile ]; then \
			echo "$(BLUE)[INFO]$(NC) Building $component Docker image..."; \
			docker build -t nestor/$component:dev -f $component/Dockerfile .; \
		else \
			echo "$(YELLOW)[WARNING]$(NC) Dockerfile not found for $component"; \
		fi; \
	done
	@echo "$(GREEN)[SUCCESS]$(NC) Docker images built"

docker-up: ## Start development environment with Docker Compose
	@echo "$(BLUE)[INFO]$(NC) Starting Docker Compose environment..."
	@if [ ! -f $(DOCKER_COMPOSE_FILE) ]; then \
		echo "$(YELLOW)[WARNING]$(NC) docker-compose.yml not found, creating basic setup..."; \
		make create-docker-compose; \
	fi
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d
	@echo "$(GREEN)[SUCCESS]$(NC) Development environment started"
	@echo "$(CYAN)[INFO]$(NC) Services available:"
	@echo "  - DynamoDB Local: http://localhost:8000"
	@echo "  - Redis: localhost:6379"

docker-up-test: ## Start test environment (minimal services)
	@echo "$(BLUE)[INFO]$(NC) Starting test environment..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d dynamodb redis
	@echo "$(GREEN)[SUCCESS]$(NC) Test environment started"

docker-down: ## Stop Docker Compose environment
	@echo "$(BLUE)[INFO]$(NC) Stopping Docker Compose environment..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down
	@echo "$(GREEN)[SUCCESS]$(NC) Development environment stopped"

docker-down-test: ## Stop test environment
	@echo "$(BLUE)[INFO]$(NC) Stopping test environment..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down
	@echo "$(GREEN)[SUCCESS]$(NC) Test environment stopped"

docker-logs: ## Show Docker Compose logs
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

docker-clean: ## Clean Docker images and containers
	@echo "$(BLUE)[INFO]$(NC) Cleaning Docker resources..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down -v --remove-orphans
	@docker image prune -f
	@docker container prune -f
	@echo "$(GREEN)[SUCCESS]$(NC) Docker cleanup completed"

create-docker-compose: ## Create basic docker-compose.yml
	@cat > $(DOCKER_COMPOSE_FILE) << 'EOF'
version: '3.8'

services:
  dynamodb:
    image: amazon/dynamodb-local:2.0.0
    container_name: nestor-dynamodb
    command: ["-jar", "DynamoDBLocal.jar", "-sharedDb", "-optimizeDbBeforeStartup"]
    ports:
      - "8000:8000"
    volumes:
      - dynamodb-data:/home/dynamodblocal/data
    working_dir: /home/dynamodblocal
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: nestor-redis
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  dynamodb-data:
  redis-data:
EOF

##@ Dependencies

deps: check-workspace ## Update dependencies for all components
	@echo "$(BLUE)[INFO]$(NC) Updating dependencies..."
	@go work sync
	@for dir in $(CLI_DIR) $(ORCHESTRATOR_DIR) $(PROCESSOR_DIR) $(SHARED_DIR); do \
		echo "$(BLUE)[INFO]$(NC) Updating dependencies for $dir..."; \
		cd $dir && go mod tidy && cd ..; \
	done
	@echo "$(GREEN)[SUCCESS]$(NC) Dependencies updated"

deps-upgrade: check-workspace ## Upgrade dependencies for all components
	@echo "$(BLUE)[INFO]$(NC) Upgrading dependencies..."
	@for dir in $(CLI_DIR) $(ORCHESTRATOR_DIR) $(PROCESSOR_DIR) $(SHARED_DIR); do \
		echo "$(BLUE)[INFO]$(NC) Upgrading dependencies for $dir..."; \
		cd $dir && go get -u ./... && go mod tidy && cd ..; \
	done
	@go work sync
	@echo "$(GREEN)[SUCCESS]$(NC) Dependencies upgraded"

deps-verify: check-workspace ## Verify dependencies are consistent
	@echo "$(BLUE)[INFO]$(NC) Verifying dependencies..."
	@go mod verify
	@echo "$(GREEN)[SUCCESS]$(NC) Dependencies verified"

deps-graph: check-workspace ## Show dependency graph (requires graphviz)
	@echo "$(BLUE)[INFO]$(NC) Generating dependency graph..."
	@if command -v dot >/dev/null 2>&1; then \
		go mod graph | modgraphviz | dot -Tpng -o $(LOGS_DIR)/deps.png; \
		echo "$(GREEN)[SUCCESS]$(NC) Dependency graph saved to $(LOGS_DIR)/deps.png"; \
	else \
		echo "$(YELLOW)[WARNING]$(NC) graphviz not installed. Install with: brew install graphviz (macOS)"; \
	fi

##@ Releases

release-cli: check-workspace ## Tag and release CLI component
	@echo "$(BLUE)[INFO]$(NC) Releasing CLI component..."
	@./scripts/release.sh cli

release-orchestrator: check-workspace ## Tag and release Orchestrator component
	@echo "$(BLUE)[INFO]$(NC) Releasing Orchestrator component..."
	@./scripts/release.sh orchestrator

release-processor: check-workspace ## Tag and release Processor component
	@echo "$(BLUE)[INFO]$(NC) Releasing Processor component..."
	@./scripts/release.sh processor

release-shared: check-workspace ## Tag and release Shared libraries
	@echo "$(BLUE)[INFO]$(NC) Releasing Shared libraries..."
	@./scripts/release.sh shared

versions: ## Show current version of all components
	@echo "$(BLUE)[INFO]$(NC) Component Versions:"
	@echo "$(CYAN)CLI:$(NC)          $(git tag -l 'cli/v*' | sort -V | tail -1 | sed 's/cli\///' || echo 'v0.0.0-dev')"
	@echo "$(CYAN)Orchestrator:$(NC) $(git tag -l 'orchestrator/v*' | sort -V | tail -1 | sed 's/orchestrator\///' || echo 'v0.0.0-dev')"
	@echo "$(CYAN)Processor:$(NC)    $(git tag -l 'processor/v*' | sort -V | tail -1 | sed 's/processor\///' || echo 'v0.0.0-dev')"
	@echo "$(CYAN)Shared:$(NC)       $(git tag -l 'shared/v*' | sort -V | tail -1 | sed 's/shared\///' || echo 'v0.0.0-dev')"
	@echo "$(CYAN)Current:$(NC)      $(GIT_VERSION)"

release-status: ## Check status of recent releases
	@echo "$(BLUE)[INFO]$(NC) Recent Release Status:"
	@if [ -f scripts/release-status.sh ]; then \
		./scripts/release-status.sh; \
	else \
		echo "$(YELLOW)[WARNING]$(NC) release-status.sh not found"; \
	fi

##@ Code Generation

generate: check-workspace ## Run code generation
	@echo "$(BLUE)[INFO]$(NC) Running code generation..."
	@go generate ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Code generation completed"

mocks: check-workspace ## Generate mocks for testing
	@echo "$(BLUE)[INFO]$(NC) Generating mocks..."
	@if [ -f $(TOOLS_DIR)/generate/main.go ]; then \
		go run $(TOOLS_DIR)/generate/main.go mocks; \
	else \
		echo "$(YELLOW)[WARNING]$(NC) Mock generator not found at $(TOOLS_DIR)/generate/main.go"; \
	fi
	@echo "$(GREEN)[SUCCESS]$(NC) Mocks generated"

##@ Cleanup

clean: ## Clean build artifacts and cache
	@echo "$(BLUE)[INFO]$(NC) Cleaning build artifacts..."
	@rm -rf $(DIST_DIR)
	@rm -rf $(COVERAGE_DIR)
	@rm -rf $(LOGS_DIR)
	@go clean -cache
	@echo "$(GREEN)[SUCCESS]$(NC) Cleanup completed"

clean-mods: ## Clean module cache
	@echo "$(BLUE)[INFO]$(NC) Cleaning module cache..."
	@go clean -modcache
	@echo "$(GREEN)[SUCCESS]$(NC) Module cache cleaned"

clean-docker: docker-clean ## Clean Docker resources

clean-all: clean clean-mods clean-docker ## Clean everything (build artifacts, cache, Docker)
	@echo "$(GREEN)[SUCCESS]$(NC) Full cleanup completed"

##@ Checks & Validation

check: check-workspace lint test security ## Run all quality checks (lint, test, security)
	@echo "$(GREEN)[SUCCESS]$(NC) All quality checks passed! 🎉"

validate: check-workspace deps-verify fmt vet ## Run all validation checks
	@echo "$(GREEN)[SUCCESS]$(NC) All validation checks passed!"

ci-local: setup check build-all-platforms ## Run CI checks locally (same as GitHub Actions)
	@echo "$(GREEN)[SUCCESS]$(NC) Local CI checks completed successfully! 🚀"

pre-commit: fmt lint test ## Run pre-commit checks
	@echo "$(GREEN)[SUCCESS]$(NC) Pre-commit checks passed!"

##@ Utilities

info: ## Show development environment information
	@echo "$(CYAN)═══════════════════════════════════════$(NC)"
	@echo "$(CYAN)    Nestor Development Environment     $(NC)"
	@echo "$(CYAN)═══════════════════════════════════════$(NC)"
	@echo "$(YELLOW)Go Version:$(NC)        $(go version 2>/dev/null || echo 'Not installed')"
	@echo "$(YELLOW)Required:$(NC)          $(REQUIRED_GO_VERSION)+"
	@echo "$(YELLOW)Workspace:$(NC)         $(if [ -f $(WORKSPACE_FILE) ]; then echo "✅ $(WORKSPACE_FILE)"; else echo "❌ Not found"; fi)"
	@echo "$(YELLOW)Git Commit:$(NC)        $(GIT_COMMIT)"
	@echo "$(YELLOW)Git Version:$(NC)       $(GIT_VERSION)"
	@echo "$(YELLOW)Build Date:$(NC)        $(BUILD_DATE)"
	@echo "$(CYAN)───────────────────────────────────────$(NC)"
	@echo "$(YELLOW)Tools:$(NC)"
	@echo "  Docker:               $(if command -v docker >/dev/null 2>&1; then echo "✅ $(docker --version | cut -d' ' -f3 | tr -d ',')"; else echo "❌ Not installed"; fi)"
	@echo "  Node.js:              $(if command -v node >/dev/null 2>&1; then echo "✅ $(node --version)"; else echo "❌ Not installed"; fi)"
	@echo "  golangci-lint:        $(if command -v golangci-lint >/dev/null 2>&1; then echo "✅ Available"; else echo "❌ Not installed"; fi)"
	@echo "  gosec:                $(if command -v gosec >/dev/null 2>&1; then echo "✅ Available"; else echo "❌ Not installed"; fi)"
	@echo "  goreleaser:           $(if command -v goreleaser >/dev/null 2>&1; then echo "✅ Available"; else echo "❌ Not installed"; fi)"
	@echo "  air:                  $(if command -v air >/dev/null 2>&1; then echo "✅ Available"; else echo "❌ Not installed"; fi)"
	@echo "  entr:                 $(if command -v entr >/dev/null 2>&1; then echo "✅ Available"; else echo "❌ Not installed"; fi)"
	@echo "$(CYAN)═══════════════════════════════════════$(NC)"

watch: check-workspace ## Watch for file changes and run tests
	@echo "$(BLUE)[INFO]$(NC) Watching for file changes..."
	@if command -v entr >/dev/null 2>&1; then \
		find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" -not -path "./$(DIST_DIR)/*" | entr -c make test-all; \
	else \
		echo "$(RED)[ERROR]$(NC) entr not installed. Install with: brew install entr (macOS) or apt install entr (Ubuntu)"; \
		exit 1; \
	fi

benchmark: check-workspace ## Run benchmarks for all components
	@echo "$(BLUE)[INFO]$(NC) Running benchmarks..."
	@for dir in $(CLI_DIR) $(ORCHESTRATOR_DIR) $(PROCESSOR_DIR) $(SHARED_DIR); do \
		echo "$(BLUE)[INFO]$(NC) Running benchmarks for $dir..."; \
		cd $dir && go test -bench=. -benchmem ./... && cd ..; \
	done
	@echo "$(GREEN)[SUCCESS]$(NC) Benchmarks completed"

# Include component-specific Makefiles if they exist
-include $(CLI_DIR)/Makefile.local
-include $(ORCHESTRATOR_DIR)/Makefile.local
-include $(PROCESSOR_DIR)/Makefile.local
