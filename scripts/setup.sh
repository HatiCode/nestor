#!/bin/bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Component directories
CLI_DIR="$ROOT_DIR/cli"
ORCHESTRATOR_DIR="$ROOT_DIR/orchestrator"
PROCESSOR_DIR="$ROOT_DIR/processor"
SHARED_DIR="$ROOT_DIR/shared"
TOOLS_DIR="$ROOT_DIR/tools"
DOCS_DIR="$ROOT_DIR/docs"

# Build directories
DIST_DIR="$ROOT_DIR/dist"
COVERAGE_DIR="$ROOT_DIR/coverage"
LOGS_DIR="$ROOT_DIR/logs"

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_dependencies() {
    log_info "Checking system dependencies..."

    # Check Go
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is not installed. Please install Go 1.24+ from https://golang.org/dl/"
        exit 1
    fi

    # Check Go version
    go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
    required_version="go1.24"
    if [[ "$(printf '%s\n' "$required_version" "$go_version" | sort -V | head -n1)" != "$required_version" ]]; then
        log_error "Go version $go_version is too old. Please upgrade to Go 1.24+"
        exit 1
    fi
    log_success "Go version: $go_version"

    # Check Git
    if ! command -v git >/dev/null 2>&1; then
        log_error "Git is not installed. Please install Git from https://git-scm.com/"
        exit 1
    fi
    log_success "Git: $(git --version)"

    # Check Docker (optional but recommended)
    if command -v docker >/dev/null 2>&1; then
        log_success "Docker: $(docker --version | head -1)"
    else
        log_warning "Docker not found. Some development features will be unavailable."
    fi

    # Check Node.js (for docs)
    if command -v node >/dev/null 2>&1; then
        log_success "Node.js: $(node --version)"
    else
        log_warning "Node.js not found. Documentation features will be unavailable."
    fi
}

install_dev_tools() {
    log_info "Installing development tools..."

    # Install golangci-lint
    if ! command -v golangci-lint >/dev/null 2>&1; then
        log_info "Installing golangci-lint..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(go env GOPATH)/bin" latest
        log_success "golangci-lint installed"
    else
        log_success "golangci-lint already installed: $(golangci-lint --version)"
    fi

    # Install gosec
    if ! command -v gosec >/dev/null 2>&1; then
        log_info "Installing gosec..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
        log_success "gosec installed"
    else
        log_success "gosec already installed"
    fi

    # Install goreleaser
    if ! command -v goreleaser >/dev/null 2>&1; then
        log_info "Installing goreleaser..."
        go install github.com/goreleaser/goreleaser@latest
        log_success "goreleaser installed"
    else
        log_success "goreleaser already installed: $(goreleaser --version | head -1)"
    fi

    # Install trivy
    if ! command -v trivy >/dev/null 2>&1; then
        log_info "Installing trivy..."
        go install github.com/aquasecurity/trivy@latest
        log_success "trivy installed"
    else
        log_success "trivy already installed"
    fi

    # Install useful tools
    local tools=(
        "github.com/air-verse/air@latest"              # Live reload for development
        "github.com/golangci/misspell/cmd/misspell@latest" # Spell checker
        "github.com/client9/misspell/cmd/misspell@latest"  # Alternative spell checker
    )

    for tool in "${tools[@]}"; do
        tool_name=$(basename "$(echo "$tool" | cut -d'@' -f1)")
        if ! command -v "$tool_name" >/dev/null 2>&1; then
            log_info "Installing $tool_name..."
            go install "$tool" || log_warning "Failed to install $tool_name"
        fi
    done
}

create_directories() {
    log_info "Creating necessary directories..."

    local dirs=(
        "$DIST_DIR"
        "$COVERAGE_DIR"
        "$LOGS_DIR"
        "$TOOLS_DIR"
        "$TOOLS_DIR/lint"
        "$TOOLS_DIR/generate"
        "$ROOT_DIR/.github/scripts"
        "$ROOT_DIR/test/integration"
        "$ROOT_DIR/deployments/helm"
        "$ROOT_DIR/deployments/docker"
        "$ROOT_DIR/deployments/k8s"
    )

    for dir in "${dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            mkdir -p "$dir"
            log_success "Created directory: $dir"
        fi
    done
}

setup_go_workspace() {
    log_info "Setting up Go workspace..."

    cd "$ROOT_DIR"

    # Initialize or sync workspace
    if [[ ! -f "go.work" ]]; then
        log_info "Initializing Go workspace..."
        go work init "$CLI_DIR" "$ORCHESTRATOR_DIR" "$PROCESSOR_DIR" "$SHARED_DIR"
        log_success "Go workspace initialized"
    else
        log_info "Syncing Go workspace..."
        go work sync
        log_success "Go workspace synchronized"
    fi

    # Verify workspace
    go work verify
    log_success "Go workspace verified"
}

create_component_structure() {
    log_info "Creating component directory structures..."

    # CLI structure
    if [[ ! -d "$CLI_DIR" ]]; then
        mkdir -p "$CLI_DIR"/{cmd,internal/{config,version},pkg}
        touch "$CLI_DIR"/{main.go,README.md}
        log_success "Created CLI directory structure"
    fi

    # Orchestrator structure (following the README.md structure)
    if [[ ! -d "$ORCHESTRATOR_DIR/internal" ]]; then
        mkdir -p "$ORCHESTRATOR_DIR"/{internal/{api,catalog,deployment,dependencies,gitops,events/sse,storage,teams,policies,observability},pkg/{api,models,events},configs,deployments,examples,docs}
        touch "$ORCHESTRATOR_DIR"/{main.go,README.md}
        log_success "Created Orchestrator directory structure"
    fi

    # Processor structure
    if [[ ! -d "$PROCESSOR_DIR/internal" ]]; then
        mkdir -p "$PROCESSOR_DIR"/{internal/{config,processor,handlers},pkg,configs}
        touch "$PROCESSOR_DIR"/{main.go,README.md}
        log_success "Created Processor directory structure"
    fi

    # Shared structure
    if [[ ! -d "$SHARED_DIR/pkg" ]]; then
        mkdir -p "$SHARED_DIR"/pkg/{logging,config,errors,types,utils}
        touch "$SHARED_DIR"/README.md
        log_success "Created Shared directory structure"
    fi
}

create_config_files() {
    log_info "Creating configuration files..."

    # golangci-lint config
    if [[ ! -f "$TOOLS_DIR/lint/.golangci.yml" ]]; then
        cat > "$TOOLS_DIR/lint/.golangci.yml" << 'EOF'
run:
  timeout: 5m
  modules-download-mode: readonly

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

linters-settings:
  gci:
    sections:
      - standard
      - default
      - prefix(github.com/HatiCode/nestor)
  revive:
    min-confidence: 0
  gocritic:
    enabled-tags:
      - diagnostic
      - experimental
      - opinionated
      - performance
      - style
  gocyclo:
    min-complexity: 15

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gosec
        - dupl
    - path: main\.go
      linters:
        - gocyclo
EOF
        log_success "Created golangci-lint configuration"
    fi

    # Docker compose for development
    if [[ ! -f "$ROOT_DIR/docker-compose.yml" ]]; then
        cat > "$ROOT_DIR/docker-compose.yml" << 'EOF'
version: '3.8'

services:
  dynamodb:
    image: amazon/dynamodb-local:latest
    container_name: nestor-dynamodb
    command: ["-jar", "DynamoDBLocal.jar", "-sharedDb", "-optimizeDbBeforeStartup"]
    ports:
      - "8000:8000"
    volumes:
      - dynamodb-data:/home/dynamodblocal/data
    working_dir: /home/dynamodblocal

  redis:
    image: redis:7-alpine
    container_name: nestor-redis
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data

  orchestrator:
    build:
      context: .
      dockerfile: orchestrator/Dockerfile
    container_name: nestor-orchestrator
    ports:
      - "8080:8080"
    environment:
      - DYNAMODB_ENDPOINT=http://dynamodb:8000
      - REDIS_URL=redis://redis:6379
    depends_on:
      - dynamodb
      - redis
    volumes:
      - ./orchestrator/configs:/app/configs

volumes:
  dynamodb-data:
  redis-data:
EOF
        log_success "Created docker-compose.yml"
    fi

    # Basic .env template
    if [[ ! -f "$ROOT_DIR/.env.example" ]]; then
        cat > "$ROOT_DIR/.env.example" << 'EOF'
# Development environment variables
DYNAMODB_ENDPOINT=http://localhost:8000
REDIS_URL=redis://localhost:6379
LOG_LEVEL=debug
GITHUB_TOKEN=your_github_token_here
EOF
        log_success "Created .env.example"
    fi
}

create_go_modules() {
    log_info "Creating Go modules for components..."

    # CLI module
    if [[ ! -f "$CLI_DIR/go.mod" ]]; then
        cd "$CLI_DIR"
        go mod init github.com/HatiCode/nestor/cli
        log_success "Created CLI go.mod"
    fi

    # Orchestrator module (already exists based on provided files)
    if [[ ! -f "$ORCHESTRATOR_DIR/go.mod" ]]; then
        cd "$ORCHESTRATOR_DIR"
        go mod init github.com/HatiCode/nestor/orchestrator
        log_success "Created Orchestrator go.mod"
    fi

    # Processor module
    if [[ ! -f "$PROCESSOR_DIR/go.mod" ]]; then
        cd "$PROCESSOR_DIR"
        go mod init github.com/HatiCode/nestor/processor
        log_success "Created Processor go.mod"
    fi

    # Shared module
    if [[ ! -f "$SHARED_DIR/go.mod" ]]; then
        cd "$SHARED_DIR"
        go mod init github.com/HatiCode/nestor/shared
        log_success "Created Shared go.mod"
    fi

    cd "$ROOT_DIR"
}

setup_git_hooks() {
    log_info "Setting up Git hooks..."

    # Create pre-commit hook
    if [[ ! -f "$ROOT_DIR/.git/hooks/pre-commit" ]]; then
        cat > "$ROOT_DIR/.git/hooks/pre-commit" << 'EOF'
#!/bin/bash
set -e

echo "Running pre-commit checks..."

# Run formatting
make fmt

# Run linting
make lint

# Run tests
make test

echo "Pre-commit checks passed!"
EOF
        chmod +x "$ROOT_DIR/.git/hooks/pre-commit"
        log_success "Created pre-commit hook"
    fi

    # Create commit-msg hook for conventional commits
    if [[ ! -f "$ROOT_DIR/.git/hooks/commit-msg" ]]; then
        cat > "$ROOT_DIR/.git/hooks/commit-msg" << 'EOF'
#!/bin/bash

commit_regex='^(feat|fix|docs|style|refactor|test|chore|ci|perf|build)(\(.+\))?: .{1,50}'

if ! grep -qE "$commit_regex" "$1"; then
    echo "Invalid commit message format!"
    echo "Format: type(scope): description"
    echo "Types: feat, fix, docs, style, refactor, test, chore, ci, perf, build"
    echo "Example: feat(cli): add new generate command"
    exit 1
fi
EOF
        chmod +x "$ROOT_DIR/.git/hooks/commit-msg"
        log_success "Created commit-msg hook"
    fi
}

create_additional_scripts() {
    log_info "Creating additional helper scripts..."

    # Create release status script
    cat > "$ROOT_DIR/scripts/release-status.sh" << 'EOF'
#!/bin/bash
set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

components=("cli" "orchestrator" "processor" "shared")

for component in "${components[@]}"; do
    latest_tag=$(git tag -l "${component}/v*" | sort -V | tail -1 || echo "")
    if [[ -n "$latest_tag" ]]; then
        version=${latest_tag#${component}/}
        echo -e "${GREEN}✅${NC} ${component}: ${version}"
    else
        echo -e "${YELLOW}⚠️${NC}  ${component}: No releases yet"
    fi
done
EOF
    chmod +x "$ROOT_DIR/scripts/release-status.sh"

    # Create install hooks script
    cat > "$ROOT_DIR/scripts/install-hooks.sh" << 'EOF'
#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo "Installing Git hooks..."

# Copy hooks to .git/hooks
if [[ -d "$ROOT_DIR/.git/hooks" ]]; then
    # Enable pre-commit hook
    if [[ ! -f "$ROOT_DIR/.git/hooks/pre-commit" ]]; then
        cp "$ROOT_DIR/scripts/pre-commit" "$ROOT_DIR/.git/hooks/pre-commit"
        chmod +x "$ROOT_DIR/.git/hooks/pre-commit"
        echo "✅ Pre-commit hook installed"
    fi

    # Enable commit-msg hook
    if [[ ! -f "$ROOT_DIR/.git/hooks/commit-msg" ]]; then
        cp "$ROOT_DIR/scripts/commit-msg" "$ROOT_DIR/.git/hooks/commit-msg"
        chmod +x "$ROOT_DIR/.git/hooks/commit-msg"
        echo "✅ Commit message hook installed"
    fi
else
    echo "❌ Not a Git repository or .git/hooks directory not found"
    exit 1
fi

echo "🎉 Git hooks installed successfully!"
EOF
    chmod +x "$ROOT_DIR/scripts/install-hooks.sh"

    log_success "Created additional helper scripts"
}

run_initial_setup() {
    log_info "Running initial setup tasks..."

    cd "$ROOT_DIR"

    # Update workspace
    go work sync

    # Tidy modules
    for dir in "$CLI_DIR" "$ORCHESTRATOR_DIR" "$PROCESSOR_DIR" "$SHARED_DIR"; do
        if [[ -f "$dir/go.mod" ]]; then
            cd "$dir"
            go mod tidy
            cd "$ROOT_DIR"
        fi
    done

    # Create .gitignore additions if needed
    if ! grep -q "dist/" "$ROOT_DIR/.gitignore" 2>/dev/null; then
        echo -e "\n# Build artifacts\ndist/\ncoverage/\nlogs/" >> "$ROOT_DIR/.gitignore"
        log_success "Updated .gitignore"
    fi

    log_success "Initial setup completed"
}

print_next_steps() {
    echo
    echo -e "${CYAN}🎉 Nestor development environment setup completed!${NC}"
    echo
    echo -e "${YELLOW}Next steps:${NC}"
    echo "1. Run 'make check' to verify everything is working"
    echo "2. Run 'make build' to build all components"
    echo "3. Run 'make docker-up' to start the development environment"
    echo "4. Check out the documentation in docs/"
    echo
    echo -e "${YELLOW}Useful commands:${NC}"
    echo "  make help          - Show all available commands"
    echo "  make dev-orchestrator - Run orchestrator in development mode"
    echo "  make test          - Run all tests"
    echo "  make lint          - Run linters"
    echo
    echo -e "${YELLOW}Development tools installed:${NC}"
    echo "  golangci-lint - Code linting"
    echo "  gosec         - Security scanning"
    echo "  goreleaser    - Release automation"
    echo "  trivy         - Vulnerability scanning"
    echo
}

main() {
    echo -e "${CYAN}🚀 Setting up Nestor development environment...${NC}"
    echo

    check_dependencies
    install_dev_tools
    create_directories
    create_component_structure
    create_go_modules
    setup_go_workspace
    create_config_files
    setup_git_hooks
    create_additional_scripts
    run_initial_setup

    print_next_steps
}

# Run main function
main "$@"
