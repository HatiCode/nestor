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

# Go version management - SINGLE SOURCE OF TRUTH
GO_VERSION="1.24.4"
MIN_GO_VERSION="1.24"

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
        log_error "Go is not installed. Please install Go ${MIN_GO_VERSION}+ from https://golang.org/dl/"
        exit 1
    fi

    # Check Go version more precisely
    go_version_output=$(go version)
    go_version=$(echo "$go_version_output" | grep -oE 'go[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)

    if [[ -z "$go_version" ]]; then
        log_error "Could not determine Go version from: $go_version_output"
        exit 1
    fi

    # Version comparison function
    version_compare() {
        printf '%s\n%s\n' "$1" "$2" | sort -V | head -n1
    }

    min_version="go${MIN_GO_VERSION}"
    if [[ "$(version_compare "$min_version" "$go_version")" != "$min_version" ]]; then
        log_error "Go version $go_version is too old. Please upgrade to Go ${MIN_GO_VERSION}+"
        log_info "Current: $go_version, Required: ${MIN_GO_VERSION}+"
        exit 1
    fi

    log_success "Go version: $go_version (required: ${MIN_GO_VERSION}+)"

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

    # Ensure Go bin directory is in PATH
    go_bin_dir=$(go env GOPATH)/bin
    if [[ ! "$PATH" == *"$go_bin_dir"* ]]; then
        log_warning "Go bin directory not in PATH. Adding to current session..."
        export PATH="$PATH:$go_bin_dir"
    fi

    # Install golangci-lint
    if ! command -v golangci-lint >/dev/null 2>&1; then
        log_info "Installing golangci-lint..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$go_bin_dir" latest
        log_success "golangci-lint installed"
    else
        log_success "golangci-lint already installed: $(golangci-lint --version | head -1)"
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

    # Install useful development tools
    local tools=(
        "github.com/air-verse/air@latest"                          # Live reload for development
        "github.com/client9/misspell/cmd/misspell@latest"          # Spell checker
        "github.com/wadey/gocovmerge@latest"                       # Coverage merging
    )

    for tool in "${tools[@]}"; do
        tool_name=$(basename "$(echo "$tool" | cut -d'@' -f1)")
        if ! command -v "$tool_name" >/dev/null 2>&1; then
            log_info "Installing $tool_name..."
            if go install "$tool"; then
                log_success "$tool_name installed"
            else
                log_warning "Failed to install $tool_name"
            fi
        else
            log_success "$tool_name already installed"
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

    # Create or update workspace with proper Go version
    log_info "Creating/updating Go workspace with Go ${GO_VERSION}..."
    cat > go.work << EOF
go ${GO_VERSION}

use (
	./cli
	./orchestrator
	./processor
	./shared
)

// Replace directives for local development
// Uncomment these when working with local versions of dependencies
// replace (
//	github.com/HatiCode/nestor/shared => ./shared
// )
EOF

    # Sync and verify workspace
    log_info "Syncing Go workspace..."
    go work sync
    go work verify
    log_success "Go workspace initialized and verified"

    # Show workspace info
    log_info "Workspace information:"
    go version
    echo "Modules in workspace:"
    go list -m all | head -5
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
        log_success "Created golangci-lint configuration"
    fi

    # Docker compose for development
    if [[ ! -f "$ROOT_DIR/docker-compose.yml" ]]; then
        cat > "$ROOT_DIR/docker-compose.yml" << 'EOF'
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
      - LOG_LEVEL=debug
    depends_on:
      dynamodb:
        condition: service_healthy
      redis:
        condition: service_healthy
    volumes:
      - ./orchestrator/configs:/app/configs

volumes:
  dynamodb-data:
  redis-data:
EOF
        log_success "Created docker-compose.yml"
    fi

    # Air configuration for live reload
    if [[ ! -f "$ROOT_DIR/.air.toml" ]]; then
        cat > "$ROOT_DIR/.air.toml" << 'EOF'
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./orchestrator"
  delay = 0
  exclude_dir = ["assets", "tmp", "vendor", "testdata", "dist", "coverage", "logs"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html", "yaml", "yml"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
  keep_scroll = true
EOF
        log_success "Created .air.toml configuration"
    fi

    # Basic .env template
    if [[ ! -f "$ROOT_DIR/.env.example" ]]; then
        cat > "$ROOT_DIR/.env.example" << 'EOF'
# Development environment variables
DYNAMODB_ENDPOINT=http://localhost:8000
REDIS_URL=redis://localhost:6379
LOG_LEVEL=debug
GITHUB_TOKEN=your_github_token_here

# Component-specific settings
ORCHESTRATOR_PORT=8080
PROCESSOR_WORKERS=4
CLI_CONFIG_DIR=~/.nestor
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

    # Orchestrator module (update if exists or create)
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

    # Ensure all modules use the same Go version
    for dir in "$CLI_DIR" "$ORCHESTRATOR_DIR" "$PROCESSOR_DIR" "$SHARED_DIR"; do
        if [[ -f "$dir/go.mod" ]]; then
            log_info "Updating Go version in $dir/go.mod..."
            cd "$dir"

            # Update go directive in go.mod if it exists
            if grep -q "^go " go.mod; then
                sed -i.bak "s/^go .*/go ${GO_VERSION}/" go.mod && rm go.mod.bak
            else
                # Add go directive if it doesn't exist
                echo -e "\ngo ${GO_VERSION}" >> go.mod
            fi

            go mod tidy
            cd "$ROOT_DIR"
        fi
    done
}

setup_git_hooks() {
    log_info "Setting up Git hooks..."

    # Create pre-commit hook
    if [[ ! -f "$ROOT_DIR/.git/hooks/pre-commit" ]]; then
        cat > "$ROOT_DIR/.git/hooks/pre-commit" << 'EOF'
#!/bin/bash
set -e

echo "🔍 Running pre-commit checks..."

# Check if we're in the right directory
if [[ ! -f "Makefile" ]]; then
    echo "❌ Not in repository root directory"
    exit 1
fi

# Run formatting
echo "📝 Formatting code..."
make fmt

# Run linting
echo "🔍 Running linters..."
make lint

# Run tests
echo "🧪 Running tests..."
make test

echo "✅ Pre-commit checks passed!"
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
    echo "❌ Invalid commit message format!"
    echo ""
    echo "📋 Format: type(scope): description"
    echo "🏷️  Types: feat, fix, docs, style, refactor, test, chore, ci, perf, build"
    echo "📝 Example: feat(cli): add new generate command"
    echo ""
    echo "Your message: $(head -n1 "$1")"
    exit 1
fi

echo "✅ Commit message format is valid"
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
CYAN='\033[0;36m'
NC='\033[0m'

components=("cli" "orchestrator" "processor" "shared")

echo -e "${CYAN}📦 Release Status Overview${NC}"
echo "=========================="

for component in "${components[@]}"; do
    latest_tag=$(git tag -l "${component}/v*" | sort -V | tail -1 || echo "")
    if [[ -n "$latest_tag" ]]; then
        version=${latest_tag#${component}/}
        tag_date=$(git log -1 --format=%ai "$latest_tag")
        echo -e "${GREEN}✅${NC} ${component}: ${version} (${tag_date})"
    else
        echo -e "${YELLOW}⚠️${NC}  ${component}: No releases yet"
    fi
done

echo ""
echo -e "${CYAN}🔗 Quick Commands${NC}"
echo "=================="
echo "Release CLI:          make release-cli"
echo "Release Orchestrator: make release-orchestrator"
echo "Release Processor:    make release-processor"
echo "Release Shared:       make release-shared"
EOF
    chmod +x "$ROOT_DIR/scripts/release-status.sh"

    # Create install hooks script
    cat > "$ROOT_DIR/scripts/install-hooks.sh" << 'EOF'
#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo "🪝 Installing Git hooks..."

# Copy hooks to .git/hooks
if [[ -d "$ROOT_DIR/.git/hooks" ]]; then
    # Source the setup script to get hook creation functions
    source "$SCRIPT_DIR/setup.sh"

    # Call the hook setup function
    setup_git_hooks

    echo "🎉 Git hooks installed successfully!"
else
    echo "❌ Not a Git repository or .git/hooks directory not found"
    exit 1
fi
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
            log_info "Tidying module in $dir..."
            cd "$dir"
            go mod tidy
            cd "$ROOT_DIR"
        fi
    done

    # Update .gitignore if needed
    if ! grep -q "dist/" "$ROOT_DIR/.gitignore" 2>/dev/null; then
        cat >> "$ROOT_DIR/.gitignore" << 'EOF'

# Build artifacts
dist/
coverage/
logs/
tmp/

# Development
.env
.air.toml.local

# IDE
.idea/
.vscode/
*.swp
*.swo
*~
EOF
        log_success "Updated .gitignore"
    fi

    # Create initial integration test if it doesn't exist
    if [[ ! -f "$ROOT_DIR/test/integration/basic_test.go" ]]; then
        mkdir -p "$ROOT_DIR/test/integration"
        cat > "$ROOT_DIR/test/integration/basic_test.go" << 'EOF'
package integration

import (
	"testing"
)

func TestBasicIntegration(t *testing.T) {
	t.Log("Basic integration test placeholder")
	// TODO: Implement actual integration tests
	// This ensures the integration test directory is not empty
}
EOF
        log_success "Created basic integration test"
    fi

    log_success "Initial setup completed"
}

create_minimal_main_files() {
    log_info "Creating minimal main.go files for components..."

    # CLI main.go
    if [[ ! -f "$CLI_DIR/main.go" ]]; then
        cat > "$CLI_DIR/main.go" << 'EOF'
package main

import (
	"fmt"
	"os"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
	builtBy = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("nestor-cli %s\ncommit: %s\nbuilt at: %s\nbuilt by: %s\n", version, commit, date, builtBy)
		return
	}

	fmt.Println("🏗️  Nestor CLI - Infrastructure as Code from Code Annotations")
	fmt.Println("Usage: nestor [command]")
	fmt.Println("Commands will be implemented in future versions.")
}
EOF
        log_success "Created CLI main.go"
    fi

    # Orchestrator main.go
    if [[ ! -f "$ORCHESTRATOR_DIR/main.go" ]]; then
        cat > "$ORCHESTRATOR_DIR/main.go" << 'EOF'
package main

import (
	"fmt"
	"os"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
	builtBy = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("nestor-orchestrator %s\ncommit: %s\nbuilt at: %s\nbuilt by: %s\n", version, commit, date, builtBy)
		return
	}

	fmt.Println("🎼 Nestor Orchestrator - Central Coordination Hub")
	fmt.Println("Usage: orchestrator [command]")
	fmt.Println("Server will be implemented in future versions.")
}
EOF
        log_success "Created Orchestrator main.go"
    fi

    # Processor main.go
    if [[ ! -f "$PROCESSOR_DIR/main.go" ]]; then
        cat > "$PROCESSOR_DIR/main.go" << 'EOF'
package main

import (
	"fmt"
	"os"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
	builtBy = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("nestor-processor %s\ncommit: %s\nbuilt at: %s\nbuilt by: %s\n", version, commit, date, builtBy)
		return
	}

	fmt.Println("⚙️  Nestor Processor - Code Analysis and Infrastructure Generation")
	fmt.Println("Usage: processor [command]")
	fmt.Println("Processor will be implemented in future versions.")
}
EOF
        log_success "Created Processor main.go"
    fi
}

print_next_steps() {
    echo
    echo -e "${CYAN}🎉 Nestor development environment setup completed!${NC}"
    echo
    echo -e "${YELLOW}📋 Next steps:${NC}"
    echo "1. Run 'make check' to verify everything is working"
    echo "2. Run 'make build' to build all components"
    echo "3. Run 'make docker-up' to start the development environment"
    echo "4. Check out the documentation in docs/"
    echo
    echo -e "${YELLOW}🛠️  Useful commands:${NC}"
    echo "  make help             - Show all available commands"
    echo "  make info             - Show environment information"
    echo "  make dev-orchestrator - Run orchestrator in development mode"
    echo "  make test             - Run all tests"
    echo "  make lint             - Run linters"
    echo "  make watch            - Watch for changes and run tests"
    echo
    echo -e "${YELLOW}🔧 Development tools installed:${NC}"
    echo "  golangci-lint - Code linting and static analysis"
    echo "  gosec         - Security vulnerability scanning"
    echo "  goreleaser    - Release automation and cross-compilation"
    echo "  trivy         - Vulnerability scanner for dependencies"
    echo "  air           - Live reload for development"
    echo
    echo -e "${YELLOW}🏗️  Architecture Summary:${NC}"
    echo "  CLI           - Command-line interface for developers"
    echo "  Orchestrator  - Central coordination hub (DynamoDB + Redis)"
    echo "  Processor     - Code analysis and infrastructure generation"
    echo "  Shared        - Common libraries and utilities"
    echo
    echo -e "${YELLOW}📖 Documentation:${NC}"
    echo "  README.md              - Project overview"
    echo "  docs/                  - Detailed documentation"
    echo "  Makefile               - All available commands"
    echo "  .env.example           - Environment variables template"
    echo
    echo -e "${GREEN}✨ Ready to start developing! Happy coding! 🚀${NC}"
    echo
}

main() {
    echo -e "${CYAN}🚀 Setting up Nestor development environment...${NC}"
    echo -e "${BLUE}Go Version: ${GO_VERSION}${NC}"
    echo

    check_dependencies
    install_dev_tools
    create_directories
    create_component_structure
    create_go_modules
    setup_go_workspace
    create_config_files
    create_minimal_main_files
    setup_git_hooks
    create_additional_scripts
    run_initial_setup

    print_next_steps
}

# Error handling
trap 'log_error "Setup failed at line $LINENO. Exit code: $?"' ERR

# Run main function
main "$@"
