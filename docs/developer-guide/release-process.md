# Release Process Guide

This guide covers the complete release process for Nestor components, from planning to post-release monitoring.

## 🎯 Release Philosophy

Nestor follows **independent component releases** with these principles:

- **Semantic Versioning**: All components use [semver](https://semver.org/) strictly
- **Backward Compatibility**: Minor/patch releases never break existing APIs
- **Quality First**: Every release goes through comprehensive testing
- **Automation**: Manual steps are minimized through GitHub Actions
- **Transparency**: All changes documented with clear changelogs

## 📋 Pre-Release Checklist

### Planning Phase

- [ ] **Feature Complete**: All planned features implemented and tested
- [ ] **Documentation Updated**: README, API docs, examples reflect changes
- [ ] **Breaking Changes Identified**: If any, version bump planned accordingly
- [ ] **Dependencies Reviewed**: All dependencies up-to-date and secure
- [ ] **Cross-Component Impact**: Changes tested with dependent components

### Testing Phase

- [ ] **Unit Tests**: All tests pass locally and in CI
- [ ] **Integration Tests**: Cross-component functionality verified
- [ ] **Security Scan**: No new vulnerabilities introduced
- [ ] **Performance**: No significant performance regressions
- [ ] **Documentation Tests**: All code examples in docs work

### Release Preparation

- [ ] **Changelog Draft**: Release notes prepared
- [ ] **Version Decision**: Major/minor/patch determined
- [ ] **Release Branch**: Created if needed for release candidates
- [ ] **Final Review**: Code review completed by component owners

## 🚀 Release Types

### 1. Patch Releases (v1.0.1)

**When:** Bug fixes, security patches, documentation updates

**Process:**
```bash
# 1. Create branch for patch (optional for small fixes)
git checkout -b fix/critical-bug
# Make fixes...
git commit -m "fix: resolve memory leak in processor"

# 2. Create PR and get approval
git push origin fix/critical-bug
# Create PR, get reviews, merge to main

# 3. Tag and release
git checkout main
git pull origin main
git tag processor/v1.0.1
git push origin processor/v1.0.1
```

**Timeline:** Same day for critical fixes, within 1-2 days otherwise

### 2. Minor Releases (v1.1.0)

**When:** New features, new APIs, new deployment options

**Process:**
```bash
# 1. Feature development (usually in feature branches)
git checkout -b feature/sse-improvements
# Implement feature...
git commit -m "feat: add connection pooling to SSE server"

# 2. Integration testing
make test-integration

# 3. Documentation updates
# Update README, API docs, examples

# 4. Create PR and thorough review
git push origin feature/sse-improvements
# Create PR, get multiple reviews, update based on feedback

# 5. Release
git checkout main
git pull origin main
git tag orchestrator/v1.1.0
git push origin orchestrator/v1.1.0
```

**Timeline:** 1-2 weeks for planning, development, and testing

### 3. Major Releases (v2.0.0)

**When:** Breaking changes, architecture changes, API redesigns

**Process:**
```bash
# 1. Create release branch for extended development
git checkout -b release/v2.0.0
# Multiple feature branches merge to release branch

# 2. Extended testing period
# Alpha releases for early feedback
git tag cli/v2.0.0-alpha.1
git push origin cli/v2.0.0-alpha.1

# Beta releases for final testing
git tag cli/v2.0.0-beta.1
git push origin cli/v2.0.0-beta.1

# 3. Migration guide preparation
# Document all breaking changes
# Provide migration examples

# 4. Final release
git checkout main
git merge release/v2.0.0
git tag cli/v2.0.0
git push origin cli/v2.0.0
```

**Timeline:** 1-3 months including alpha/beta phases

## 🔢 Version Number Guidelines

### Semantic Versioning Rules

```
MAJOR.MINOR.PATCH (e.g., 2.1.3)
```

#### MAJOR (Breaking Changes)
- API changes that break backward compatibility
- Removal of deprecated features
- Configuration format changes
- Database schema changes requiring migration

**Examples:**
- `orchestrator/v2.0.0`: New storage interface (DynamoDB → pluggable)
- `cli/v3.0.0`: Command structure redesign
- `processor/v2.0.0`: New composition engine architecture

#### MINOR (New Features)
- New features that don't break existing functionality
- New API endpoints or functions
- New deployment targets (e.g., Azure Functions support)
- Deprecation warnings (but not removal)

**Examples:**
- `orchestrator/v1.1.0`: Add SSE support for real-time updates
- `processor/v1.2.0`: Add team policy validation
- `cli/v1.3.0`: Add `nestor status` command

#### PATCH (Bug Fixes)
- Bug fixes that don't change APIs
- Security patches
- Documentation fixes
- Performance improvements without API changes

**Examples:**
- `orchestrator/v1.0.1`: Fix memory leak in catalog sync
- `processor/v1.1.1`: Fix validation error messages
- `cli/v1.2.1`: Fix Windows path handling

### Pre-release Versions

```
v1.0.0-alpha.1    # Early development, may have breaking changes
v1.0.0-beta.1     # Feature complete, API stable, testing phase
v1.0.0-rc.1       # Release candidate, production ready
```

## 📝 Changelog Guidelines

### Format

Use [Keep a Changelog](https://keepachangelog.com/) format:

```markdown
# Changelog

## [Unreleased]

### Added
- New features go here

### Changed
- Changes to existing functionality

### Deprecated
- Features marked for removal

### Removed
- Features removed in this version

### Fixed
- Bug fixes

### Security
- Security improvements

## [1.2.0] - 2024-01-15

### Added
- SSE support for real-time catalog updates
- Team-based resource filtering in API

### Fixed
- Memory leak in catalog synchronization
- Race condition in dependency validation

## [1.1.0] - 2024-01-01
...
```

### Writing Good Changelog Entries

**✅ Good Examples:**
```markdown
### Added
- SSE endpoint for real-time processor notifications (#123)
- Support for Azure Functions deployment in processor (#145)
- Team quota validation in resource requests (#156)

### Fixed
- Memory leak in catalog synchronization affecting long-running deployments (#134)
- Race condition in dependency validation causing intermittent failures (#142)
```

**❌ Bad Examples:**
```markdown
### Added
- New stuff
- Various improvements

### Fixed
- Bug fixes
- Some issues
```

## 🤖 Automated Release Process

### GitHub Actions Workflow

When you push a tag (e.g., `cli/v1.2.3`), here's what happens automatically:

1. **Validation**
   - ✅ All tests must pass
   - ✅ Security scans must be clean
   - ✅ Build must succeed for all platforms

2. **Asset Creation**
   - 📦 Cross-platform binaries (Linux, macOS, Windows)
   - 🐳 Docker images (if applicable)
   - 📋 Package manager files (Homebrew, Scoop, etc.)
   - 🗜️ Lambda/serverless deployment packages (processor only)

3. **Distribution**
   - 🏷️ GitHub release with changelog
   - 📤 Container registry publishing
   - 🍺 Homebrew formula updates
   - 📦 Package repository updates

4. **Notification**
   - 💬 Slack notifications to team channels
   - 📧 Email notifications to maintainers
   - 🔗 GitHub release announcement

### Manual Steps (Minimal)

The only manual steps are:

1. **Create the tag** with proper naming
2. **Monitor the release** for any failures
3. **Verify distribution** (test downloads, installations)
4. **Announce the release** (blog posts, social media, etc.)

## 🔍 Release Validation

### Automated Checks

Every release automatically verifies:

- [ ] **All tests pass** across all supported platforms
- [ ] **Security scans clean** (no new vulnerabilities)
- [ ] **Builds successful** for all target platforms
- [ ] **Docker images work** and pass health checks
- [ ] **Installation packages valid** (can be installed and run)

### Manual Verification

After automated release, manually verify:

- [ ] **Download and test** the main binary for your platform
- [ ] **Install via package manager** (Homebrew, Docker, etc.)
- [ ] **Run basic functionality test** with the new version
- [ ] **Check release page** for correct changelog and assets
- [ ] **Verify container images** are available and working

### Verification Commands

```bash
# Test CLI release
brew install nestor-cli
nestor version
nestor generate --help

# Test Orchestrator Docker image
docker run --rm ghcr.io/nestor/nestor/orchestrator:v1.2.0 --version

# Test Processor Lambda package
unzip nestor-processor-lambda_v1.0.0.zip
./bootstrap --help
```

## 🚨 Emergency Releases

### Hotfix Process

For critical security issues or major bugs:

1. **Immediate Response** (< 2 hours)
   ```bash
   # Create hotfix branch from last release tag
   git checkout -b hotfix/security-patch processor/v1.0.0

   # Apply minimal fix
   git commit -m "security: patch critical vulnerability CVE-2024-xxxx"

   # Emergency review (can be async)
   git push origin hotfix/security-patch
   ```

2. **Fast-Track Release** (< 24 hours)
   ```bash
   # Merge to main after emergency review
   git checkout main
   git merge hotfix/security-patch

   # Release immediately
   git tag processor/v1.0.1
   git push origin processor/v1.0.1
   ```

3. **Communication**
   - 🚨 Immediate notification to users
   - 📋 Security advisory if needed
   - 📝 Post-mortem for process improvement

### Emergency Contacts

- **Security Issues**: security@nestor.dev
- **Critical Bugs**: @nestor/platform-team
- **Release Issues**: @nestor/release-team

## 📊 Post-Release Activities

### Monitoring (First 24 Hours)

- [ ] **Download Metrics**: Track adoption of new release
- [ ] **Error Reporting**: Monitor for new issues via logs/metrics
- [ ] **User Feedback**: Watch GitHub issues and community channels
- [ ] **Rollback Readiness**: Be prepared to advise rollback if needed

### Follow-up (First Week)

- [ ] **Usage Analytics**: Review how new features are being used
- [ ] **Performance Impact**: Check if release affects system performance
- [ ] **Documentation**: Update any missing docs based on user questions
- [ ] **Next Release Planning**: Incorporate learnings into next cycle

### Communication

```markdown
# Example Release Announcement

🚀 **Nestor Orchestrator v1.2.0 Released!**

## ✨ What's New
- Real-time processor notifications via Server-Sent Events
- 50% faster catalog synchronization
- Azure Functions support for processors

## 📥 How to Update
```bash
# Docker
docker pull ghcr.io/nestor/nestor/orchestrator:v1.2.0

# Binary
wget https://github.com/nestor/nestor/releases/download/orchestrator/v1.2.0/...
```

## 🔗 Links
- [Release Notes](https://github.com/nestor/nestor/releases/tag/orchestrator/v1.2.0)
- [Migration Guide](https://docs.nestor.dev/migration/v1.2.0)
- [Changelog](https://github.com/nestor/nestor/blob/main/orchestrator/CHANGELOG.md)
```

## 🛠️ Tools and Scripts

### Release Helper Scripts

```bash
# Create release (in scripts/release.sh)
./scripts/release.sh orchestrator minor

# This script:
# 1. Determines next version number
# 2. Updates CHANGELOG.md
# 3. Creates tag
# 4. Pushes tag to trigger release
```

### Version Management

```bash
# Check component versions
make versions

# Output:
# cli: v1.2.3
# orchestrator: v2.1.0
# processor: v1.5.2
# shared: v1.1.0
```

### Release Status

```bash
# Check release status
make release-status

# Output:
# ✅ cli/v1.2.3: Released successfully
# 🔄 orchestrator/v2.1.0: Build in progress
# ❌ processor/v1.5.2: Build failed
```

## 📚 References

### Internal Documentation
- [CI/CD Pipeline](./ci-cd-pipeline.md)
- [Contributing Guide](../../CONTRIBUTING.md)
- [Architecture Decisions](../architecture/decisions/)

### External Resources
- [Semantic Versioning](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/)
- [GitHub Releases](https://docs.github.com/en/repositories/releasing-projects-on-github)
- [GoReleaser Documentation](https://goreleaser.com/)

### Community
- **Discord**: #releases channel
- **GitHub Discussions**: Release announcements and feedback
- **Weekly Sync**: Tuesday releases review meeting

---

**Need help with a release?** Tag `@nestor/release-team` in any GitHub issue or reach out in the `#releases` Discord channel.
