#!/bin/bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Valid components
VALID_COMPONENTS=("cli" "orchestrator" "processor" "shared")

# Valid release types
VALID_TYPES=("major" "minor" "patch")

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

show_usage() {
    echo "Usage: $0 <component> [release_type]"
    echo
    echo "Components:"
    printf "  %s\n" "${VALID_COMPONENTS[@]}"
    echo
    echo "Release types (optional):"
    printf "  %s\n" "${VALID_TYPES[@]}"
    echo
    echo "Examples:"
    echo "  $0 cli minor          # Release CLI with minor version bump"
    echo "  $0 orchestrator       # Release orchestrator (will prompt for type)"
    echo "  $0 processor patch    # Release processor with patch version bump"
    echo
    echo "The script will:"
    echo "  1. Validate the current state (clean working directory, tests pass)"
    echo "  2. Determine the next version number"
    echo "  3. Update CHANGELOG.md (if it exists)"
    echo "  4. Create and push the release tag"
    echo "  5. GitHub Actions will handle the actual release"
}

validate_component() {
    local component="$1"

    for valid in "${VALID_COMPONENTS[@]}"; do
        if [[ "$component" == "$valid" ]]; then
            return 0
        fi
    done

    log_error "Invalid component: $component"
    echo "Valid components: ${VALID_COMPONENTS[*]}"
    exit 1
}

validate_release_type() {
    local release_type="$1"

    for valid in "${VALID_TYPES[@]}"; do
        if [[ "$release_type" == "$valid" ]]; then
            return 0
        fi
    done

    log_error "Invalid release type: $release_type"
    echo "Valid types: ${VALID_TYPES[*]}"
    exit 1
}

check_working_directory() {
    log_info "Checking working directory state..."

    # Check if we're in a git repository
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        log_error "Not in a Git repository"
        exit 1
    fi

    # Check for uncommitted changes
    if ! git diff --quiet || ! git diff --staged --quiet; then
        log_error "Working directory has uncommitted changes"
        log_info "Please commit or stash your changes before releasing"
        exit 1
    fi

    # Check if we're on main branch
    current_branch=$(git branch --show-current)
    if [[ "$current_branch" != "main" ]]; then
        log_warning "Not on main branch (current: $current_branch)"
        read -p "Continue anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Aborting release"
            exit 1
        fi
    fi

    # Ensure we have the latest changes
    log_info "Fetching latest changes..."
    git fetch origin

    # Check if we're behind origin
    if [[ "$(git rev-list HEAD..origin/main --count)" -gt 0 ]]; then
        log_error "Local branch is behind origin/main"
        log_info "Please pull the latest changes: git pull origin main"
        exit 1
    fi

    log_success "Working directory is clean and up to date"
}

get_current_version() {
    local component="$1"
    local latest_tag

    # Get the latest tag for this component
    latest_tag=$(git tag -l "${component}/v*" | sort -V | tail -1 || echo "")

    if [[ -n "$latest_tag" ]]; then
        echo "${latest_tag#${component}/v}"
    else
        echo "0.0.0"
    fi
}

bump_version() {
    local current="$1"
    local type="$2"

    # Parse version components
    local major minor patch
    IFS='.' read -r major minor patch <<< "$current"

    # Handle pre-release versions (remove any suffix after -)
    patch="${patch%%-*}"

    case "$type" in
        "major")
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        "minor")
            minor=$((minor + 1))
            patch=0
            ;;
        "patch")
            patch=$((patch + 1))
            ;;
        *)
            log_error "Invalid bump type: $type"
            exit 1
            ;;
    esac

    echo "${major}.${minor}.${patch}"
}

prompt_release_type() {
    local component="$1"
    local current_version="$2"

    echo
    log_info "Current version of $component: v$current_version"
    echo
    echo "Select release type:"
    echo "  1) patch (v$(bump_version "$current_version" "patch")) - Bug fixes"
    echo "  2) minor (v$(bump_version "$current_version" "minor")) - New features"
    echo "  3) major (v$(bump_version "$current_version" "major")) - Breaking changes"
    echo

    while true; do
        read -p "Choice (1-3): " -n 1 -r
        echo
        case $REPLY in
            1) echo "patch"; return ;;
            2) echo "minor"; return ;;
            3) echo "major"; return ;;
            *) echo "Invalid choice. Please select 1, 2, or 3." ;;
        esac
    done
}

run_component_tests() {
    local component="$1"

    log_info "Running tests for $component..."

    cd "$ROOT_DIR"

    # Run component-specific tests
    if ! make "test-$component" >/dev/null 2>&1; then
        log_error "Tests failed for $component"
        log_info "Please fix failing tests before releasing"
        exit 1
    fi

    log_success "Tests passed for $component"
}

run_linting() {
    local component="$1"

    log_info "Running linting checks..."

    cd "$ROOT_DIR"

    # Run linting (this runs across the workspace)
    if ! make lint >/dev/null 2>&1; then
        log_error "Linting failed"
        log_info "Please fix linting issues before releasing"
        log_info "Run 'make lint-fix' to auto-fix some issues"
        exit 1
    fi

    log_success "Linting passed"
}

check_component_changes() {
    local component="$1"
    local current_version="$2"

    # If this is the first release, skip change check
    if [[ "$current_version" == "0.0.0" ]]; then
        log_info "First release for $component, skipping change check"
        return 0
    fi

    log_info "Checking for changes since last release..."

    local last_tag="${component}/v${current_version}"
    local component_path

    # Set component path
    case "$component" in
        "shared")
            component_path="shared/"
            ;;
        *)
            component_path="${component}/ shared/"
            ;;
    esac

    # Check if there are any changes in the component since last tag
    if ! git diff --quiet "$last_tag" HEAD -- $component_path; then
        log_success "Changes detected since $last_tag"
        return 0
    else
        log_warning "No changes detected since $last_tag"
        read -p "Continue with release anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Aborting release"
            exit 1
        fi
    fi
}

update_changelog() {
    local component="$1"
    local new_version="$2"
    local changelog_file="$ROOT_DIR/$component/CHANGELOG.md"

    # Skip if changelog doesn't exist
    if [[ ! -f "$changelog_file" ]]; then
        log_info "No CHANGELOG.md found for $component, skipping changelog update"
        return 0
    fi

    log_info "Updating CHANGELOG.md..."

    # Get commits since last release
    local last_tag="${component}/v$(get_current_version "$component")"
    local commits

    if git rev-parse "$last_tag" >/dev/null 2>&1; then
        commits=$(git log --oneline --no-merges "${last_tag}..HEAD" -- "$component/" "shared/" | head -20)
    else
        commits=$(git log --oneline --no-merges HEAD -- "$component/" "shared/" | head -20)
    fi

    if [[ -z "$commits" ]]; then
        log_info "No commits found for changelog"
        return 0
    fi

    # Create temporary changelog entry
    local temp_entry="/tmp/changelog_entry.md"
    cat > "$temp_entry" << EOF
## [${new_version}] - $(date +%Y-%m-%d)

### Changes
$(echo "$commits" | sed 's/^/- /')

EOF

    # Insert into changelog after the "Unreleased" section
    if grep -q "## \[Unreleased\]" "$changelog_file"; then
        # Insert after Unreleased section
        sed -i "/## \[Unreleased\]/r $temp_entry" "$changelog_file"
    else
        # Insert at the top after the title
        sed -i "1r $temp_entry" "$changelog_file"
    fi

    rm "$temp_entry"

    log_success "Updated CHANGELOG.md"

    # Show the user what was added
    echo
    log_info "Changelog entry:"
    head -15 "$changelog_file" | tail -10
    echo

    # Ask if they want to edit
    read -p "Would you like to edit the changelog entry? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        "${EDITOR:-vi}" "$changelog_file"
    fi

    # Stage the changelog changes
    git add "$changelog_file"
}

create_release_tag() {
    local component="$1"
    local new_version="$2"
    local tag="${component}/v${new_version}"

    log_info "Creating release tag: $tag"

    # Commit changelog changes if any
    if ! git diff --staged --quiet; then
        git commit -m "chore($component): update changelog for v$new_version"
        log_success "Committed changelog changes"
    fi

    # Create annotated tag
    local tag_message="Release $component v$new_version"

    # Add brief description based on component
    case "$component" in
        "cli")
            tag_message="$tag_message

CLI release with improved user experience and new commands."
            ;;
        "orchestrator")
            tag_message="$tag_message

Orchestrator release with enhanced resource management and deployment coordination."
            ;;
        "processor")
            tag_message="$tag_message

Processor release with improved code analysis and infrastructure generation."
            ;;
        "shared")
            tag_message="$tag_message

Shared libraries release with common utilities and types."
            ;;
    esac

    git tag -a "$tag" -m "$tag_message"
    log_success "Created tag: $tag"
}

push_release() {
    local component="$1"
    local new_version="$2"
    local tag="${component}/v${new_version}"

    log_info "Pushing release to origin..."

    # Push main branch if we committed changelog
    if ! git diff HEAD~1 --quiet 2>/dev/null; then
        git push origin "$(git branch --show-current)"
        log_success "Pushed commits to origin"
    fi

    # Push the tag
    git push origin "$tag"
    log_success "Pushed tag to origin: $tag"

    echo
    log_success "Release $tag has been created and pushed!"
    echo
    log_info "GitHub Actions will now:"
    echo "  1. Run tests and build the release"
    echo "  2. Create GitHub release with binaries"
    echo "  3. Publish Docker images (if applicable)"
    echo "  4. Update package registries"
    echo
    log_info "Monitor the release at:"
    echo "  https://github.com/HatiCode/nestor/actions"
    echo "  https://github.com/HatiCode/nestor/releases/tag/$tag"
}

check_release_prerequisites() {
    local component="$1"

    log_info "Checking release prerequisites..."

    # Check if goreleaser config exists
    local goreleaser_config="$ROOT_DIR/$component/.goreleaser-$component.yml"
    if [[ ! -f "$goreleaser_config" ]]; then
        log_warning "GoReleaser config not found: $goreleaser_config"
        log_info "Release may not work properly without GoReleaser configuration"
    fi

    # Check if GitHub workflow exists
    local workflow_file="$ROOT_DIR/.github/workflows/release-$component.yml"
    if [[ ! -f "$workflow_file" ]]; then
        log_warning "GitHub workflow not found: $workflow_file"
        log_info "Automated release may not work without GitHub Actions workflow"
    fi

    # Check if main.go exists for the component
    if [[ "$component" != "shared" ]]; then
        local main_file="$ROOT_DIR/$component/main.go"
        if [[ ! -f "$main_file" ]]; then
            log_error "main.go not found for $component: $main_file"
            log_info "Component must have a main.go file to be releasable"
            exit 1
        fi
    fi

    log_success "Prerequisites check passed"
}

show_release_summary() {
    local component="$1"
    local current_version="$2"
    local new_version="$3"
    local release_type="$4"

    echo
    echo -e "${CYAN}═══════════════════════════════════════${NC}"
    echo -e "${CYAN}           RELEASE SUMMARY              ${NC}"
    echo -e "${CYAN}═══════════════════════════════════════${NC}"
    echo -e "${YELLOW}Component:${NC}     $component"
    echo -e "${YELLOW}Type:${NC}          $release_type"
    echo -e "${YELLOW}From:${NC}          v$current_version"
    echo -e "${YELLOW}To:${NC}            v$new_version"
    echo -e "${YELLOW}Tag:${NC}           ${component}/v${new_version}"
    echo -e "${CYAN}═══════════════════════════════════════${NC}"
    echo

    read -p "Proceed with this release? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Release cancelled"
        exit 0
    fi
}

cleanup_on_error() {
    local component="$1"
    local new_version="$2"
    local tag="${component}/v${new_version}"

    log_error "Release failed, cleaning up..."

    # Remove tag if it was created
    if git tag -l | grep -q "^$tag$"; then
        git tag -d "$tag"
        log_info "Removed local tag: $tag"
    fi

    # Reset any staged changes
    if ! git diff --staged --quiet; then
        git reset HEAD
        log_info "Reset staged changes"
    fi
}

main() {
    local component="${1:-}"
    local release_type="${2:-}"

    # Show usage if no component provided
    if [[ -z "$component" ]]; then
        show_usage
        exit 1
    fi

    # Validate inputs
    validate_component "$component"

    if [[ -n "$release_type" ]]; then
        validate_release_type "$release_type"
    fi

    cd "$ROOT_DIR"

    # Pre-flight checks
    check_working_directory
    check_release_prerequisites "$component"

    # Get current version
    local current_version
    current_version=$(get_current_version "$component")

    # Check for changes
    check_component_changes "$component" "$current_version"

    # Determine release type if not provided
    if [[ -z "$release_type" ]]; then
        release_type=$(prompt_release_type "$component" "$current_version")
    fi

    # Calculate new version
    local new_version
    new_version=$(bump_version "$current_version" "$release_type")

    # Show summary and confirm
    show_release_summary "$component" "$current_version" "$new_version" "$release_type"

    # Set up error handling
    trap 'cleanup_on_error "$component" "$new_version"' ERR

    # Run quality checks
    run_linting "$component"
    run_component_tests "$component"

    # Update changelog
    update_changelog "$component" "$new_version"

    # Create and push release
    create_release_tag "$component" "$new_version"
    push_release "$component" "$new_version"

    # Clean up error handler
    trap - ERR

    log_success "Release process completed successfully! 🎉"
}

# Handle script interruption
trap 'log_error "Script interrupted"; exit 1' INT TERM

# Run main function
main "$@"
