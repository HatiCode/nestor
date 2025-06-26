# CI/CD Pipeline Documentation

This document explains the complete CI/CD pipeline setup for the Nestor monorepo, including how it enables independent component releases while maintaining unified development workflows.

## 🏗️ Architecture Overview

Nestor uses a **monorepo with independent component versioning** strategy. This means:

- **Single repository** contains all components (CLI, Orchestrator, Processor, Shared)
- **Independent releases** for each component with separate version numbers
- **Unified development** experience with shared tooling and cross-component testing
- **Smart CI** that only tests/builds components that actually changed

## 📁 Repository Structure

```
nestor/
├── .github/
│   ├── workflows/          # GitHub Actions CI/CD pipelines
│   ├── dependabot.yml      # Automated dependency updates
│   └── CODEOWNERS         # Code ownership and review assignments
├── cli/
│   ├── .goreleaser.yml     # CLI release configuration
│   ├── go.mod             # CLI module definition
│   └── ...
├── orchestrator/
│   ├── .goreleaser.yml     # Orchestrator release configuration
│   ├── go.mod             # Orchestrator module definition
│   └── ...
├── processor/
│   ├── .goreleaser.yml     # Processor release configuration
│   ├── go.mod             # Processor module definition
│   └── ...
├── shared/
│   ├── go.mod             # Shared utilities module
│   └── pkg/               # Shared libraries
├── go.work                # Go workspace configuration
├── go.work.sum            # Go workspace checksums
└── Makefile              # Unified build commands
```

## 🔄 CI/CD Workflows

### 1. Main CI Pipeline (`.github/workflows/ci.yml`)

**Triggers:** Every push to `main`/`develop` and all pull requests

**Key Features:**
- **Smart Path Detection**: Only runs tests for components that actually changed
- **Parallel Testing**: Components test simultaneously when possible
- **Go Workspace Support**: Uses `go.work` for unified dependency management
- **Integration Testing**: Cross-component tests with real dependencies (DynamoDB, Redis)
- **Multi-platform Builds**: Tests on Linux, macOS, Windows across multiple architectures

**Workflow Steps:**
1. **Changes Detection**: Determines which components were modified
2. **Go Setup**: Configures Go version from `go.work` file
3. **Linting**: Runs `golangci-lint` across all changed components
4. **Component Testing**: Parallel tests for CLI, Orchestrator, Processor, Shared
5. **Integration Testing**: End-to-end tests across components
6. **Build Matrix**: Builds binaries for all OS/architecture combinations

### 2. Component Release Workflows

#### CLI Release (`.github/workflows/release-cli.yml`)
**Trigger:** `git tag cli/v*` (e.g., `cli/v1.2.3`)

**Outputs:**
- Cross-platform binaries (Linux, macOS, Windows)
- Homebrew formula (for macOS users)
- Scoop manifest (for Windows users)
- Debian/RPM packages (for Linux users)
- GitHub release with changelogs

#### Orchestrator Release (`.github/workflows/release-orchestrator.yml`)
**Trigger:** `git tag orchestrator/v*` (e.g., `orchestrator/v2.1.0`)

**Outputs:**
- Cross-platform binaries
- Docker images (multi-arch: amd64, arm64)
- Container registry publishing (GitHub Container Registry)
- Helm chart updates
- GitHub release with changelogs

#### Processor Release (`.github/workflows/release-processor.yml`)
**Trigger:** `git tag processor/v*` (e.g., `processor/v1.5.2`)

**Outputs:**
- Cross-platform binaries
- Docker images (multi-arch)
- AWS Lambda deployment packages
- Google Cloud Functions packages
- Azure Functions packages
- GitHub release with changelogs

### 3. Security & Quality Workflows

#### Security Scanning (`.github/workflows/security.yml`)
**Triggers:** Push to main, PRs, weekly schedule

**Tools:**
- **Gosec**: Go security analyzer
- **Trivy**: Vulnerability scanner for dependencies
- **SARIF Upload**: Results integrated into GitHub Security tab

#### Documentation (`.github/workflows/docs.yml`)
**Trigger:** Changes to documentation files

**Function:** Automatically builds and deploys documentation to GitHub Pages

## 🚀 Release Process

### Releasing a Component

1. **Make Changes**: Develop features/fixes in your component
2. **Test Locally**: Run `make test-[component]` or `make check`
3. **Create PR**: Submit pull request for review
4. **Merge to Main**: After approval and CI passes
5. **Tag Release**: Create version tag for the component
6. **Automatic Release**: GitHub Actions handles the rest

```bash
# Example: Releasing CLI v1.2.3
git checkout main
git pull origin main
git tag cli/v1.2.3
git push origin cli/v1.2.3

# This automatically triggers:
# ✅ Test suite execution
# ✅ Binary compilation for all platforms
# ✅ Docker image building (if applicable)
# ✅ Package creation (Homebrew, Scoop, etc.)
# ✅ GitHub release with changelog
# ✅ Container registry publishing
```

### Version Naming Convention

- **CLI**: `cli/v1.2.3`
- **Orchestrator**: `orchestrator/v2.1.0`  
- **Processor**: `processor/v1.5.2`
- **Shared**: `shared/v1.1.0`

Each component follows [Semantic Versioning](https://semver.org/):
- **Major** (v2.0.0): Breaking changes
- **Minor** (v1.1.0): New features, backward compatible
- **Patch** (v1.0.1): Bug fixes, backward compatible

## 🔧 Development Workflow

### Setting Up Development Environment

```bash
# Clone repository
git clone https://github.com/nestor/nestor.git
cd nestor

# Setup development environment (installs tools, creates directories, etc.)
make setup

# Verify everything works
make check
```

### Daily Development Commands

```bash
# Build all components
make build

# Test everything
make test

# Test specific component
make test-orchestrator

# Run linting
make lint

# Fix linting issues automatically
make lint-fix

# Run security scans
make security

# Start local development environment
make docker-up

# Run orchestrator locally
make dev-orchestrator

# Run processor locally  
make dev-processor

# Clean build artifacts
make clean
```

### Working Across Components

Thanks to Go workspaces, you can easily work across components:

```bash
# Make changes in shared library
vim shared/pkg/logging/logger.go

# Test in orchestrator immediately (uses local shared changes)
cd orchestrator/
go test ./internal/api/

# Test everything together
cd ../
make test
```

## 📊 Monitoring & Observability

### CI/CD Metrics

- **Build Times**: Track via GitHub Actions insights
- **Test Coverage**: Uploaded to Codecov for all components
- **Security Vulnerabilities**: Monitored via GitHub Security tab
- **Dependency Updates**: Automated via Dependabot

### Release Metrics

- **Release Frequency**: Per-component release cadence
- **Download Statistics**: GitHub releases and package managers
- **Container Pulls**: Docker image usage metrics

## 🛡️ Security & Compliance

### Automated Security

- **Dependency Scanning**: Trivy scans all dependencies weekly
- **Code Security**: Gosec analyzes Go code for security issues
- **Container Security**: Docker images scanned for vulnerabilities
- **SARIF Integration**: Security results visible in GitHub interface

### Access Control

- **CODEOWNERS**: Automatic reviewer assignment based on changed files
- **Branch Protection**: Main branch requires PR reviews and CI success
- **Secrets Management**: GitHub secrets for deployment credentials
- **Least Privilege**: Each workflow has minimal required permissions

## 🔄 Dependency Management

### Automated Updates

Dependabot automatically creates PRs for:
- **Go Module Updates**: Weekly updates for all components
- **GitHub Actions**: Weekly updates for workflow dependencies
- **Docker Images**: Weekly base image updates
- **Documentation Dependencies**: npm package updates

### Update Process

1. **Dependabot Creates PR**: Automated dependency update PR
2. **CI Validation**: Full test suite runs against new dependencies
3. **Security Scan**: New dependencies scanned for vulnerabilities
4. **Review & Merge**: Team reviews and merges if tests pass
5. **Automatic Release**: Can trigger patch releases if needed

## 🚨 Troubleshooting

### Common CI Issues

#### Tests Failing on Specific Component
```bash
# Run locally to debug
cd component-name/
go test -v ./...

# Check for missing dependencies
go mod tidy
go work sync
```

#### Release Workflow Failing
```bash
# Test release locally
cd component-name/
goreleaser release --snapshot --clean

# Check for missing environment variables or secrets
```

#### Docker Build Failing
```bash
# Test Docker build locally
docker build -t test-image -f component/Dockerfile .

# Check for missing build context or files
```

### Getting Help

1. **Check Workflow Logs**: GitHub Actions tab shows detailed logs
2. **Review Recent Changes**: Compare with last successful run
3. **Local Reproduction**: Try to reproduce the issue locally
4. **Ask for Help**: Tag `@nestor/platform-team` in issues

## 📚 References

### External Documentation
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GoReleaser Documentation](https://goreleaser.com/)
- [Go Workspaces Guide](https://go.dev/doc/tutorial/workspaces)
- [Dependabot Configuration](https://docs.github.com/en/code-security/dependabot)

### Internal Documentation
- [Contributing Guide](../CONTRIBUTING.md)
- [Architecture Overview](./architecture/overview.md)
- [Development Setup](./getting-started/development.md)
- [Release Process](./developer-guide/release-process.md)

## 🎯 Key Benefits

### For Developers
- **Fast Feedback**: Only tests what changed
- **Easy Releases**: Tag and forget - automation handles the rest
- **Cross-Component Development**: Work on multiple components seamlessly
- **Quality Assurance**: Automatic linting, testing, and security scanning

### For Users
- **Independent Updates**: Update only the components you need
- **Multiple Installation Options**: Homebrew, Docker, direct download, etc.
- **Security**: All releases automatically scanned for vulnerabilities
- **Reliability**: Comprehensive testing before any release

### For Maintainers
- **Automated Maintenance**: Dependency updates and security scanning
- **Release Consistency**: Standardized release process across components
- **Quality Control**: Enforced code review and testing requirements
- **Observability**: Complete visibility into build and release process

This CI/CD setup enables rapid, safe, and independent component development while maintaining high quality standards across the entire Nestor ecosystem.