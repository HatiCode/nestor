#!/bin/bash
set -euo pipefail

# Docker build script for the orchestrator

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Script directory and paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ORCHESTRATOR_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
PROJECT_ROOT="$(dirname "$(dirname "$ORCHESTRATOR_ROOT")")"

# Build metadata
VERSION="${VERSION:-dev}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

# Image configuration
IMAGE_NAME="${IMAGE_NAME:-nestor/orchestrator}"
IMAGE_TAG="${IMAGE_TAG:-${VERSION}}"
FULL_IMAGE_NAME="${IMAGE_NAME}:${IMAGE_TAG}"

# Platform configuration (for multi-arch builds)
PLATFORMS="${PLATFORMS:-linux/amd64}"
PUSH="${PUSH:-false}"

show_usage() {
    echo "Docker build script for Nestor Orchestrator"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -t, --tag TAG        Image tag (default: ${VERSION})"
    echo "  -n, --name NAME      Image name (default: ${IMAGE_NAME})"
    echo "  --platform PLATFORM Platform for build (default: ${PLATFORMS})"
    echo "  --push              Push image to registry"
    echo "  --multi-arch        Build for multiple architectures"
    echo "  -h, --help          Show this help message"
    echo ""
    echo "Environment variables:"
    echo "  VERSION             Version string (default: dev)"
    echo "  COMMIT              Git commit hash (auto-detected)"
    echo "  BUILD_DATE          Build timestamp (auto-generated)"
    echo "  IMAGE_NAME          Docker image name"
    echo "  IMAGE_TAG           Docker image tag"
    echo "  PLATFORMS           Build platforms"
    echo ""
    echo "Examples:"
    echo "  $0                           # Build dev image"
    echo "  $0 -t v1.0.0                # Build with specific tag"
    echo "  $0 --push                   # Build and push to registry"
    echo "  $0 --multi-arch             # Build for multiple architectures"
}

build_single_arch() {
    log_info "Building Docker image: ${FULL_IMAGE_NAME}"
    log_info "Build context: ${PROJECT_ROOT}"
    log_info "Platform: ${PLATFORMS}"

    cd "${PROJECT_ROOT}"

    docker build \
        --file orchestrator/deployments/docker/Dockerfile \
        --tag "${FULL_IMAGE_NAME}" \
        --platform "${PLATFORMS}" \
        --build-arg VERSION="${VERSION}" \
        --build-arg COMMIT="${COMMIT}" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        .

    log_success "Docker image built: ${FULL_IMAGE_NAME}"

    # Show image info
    docker images "${IMAGE_NAME}" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
}

build_multi_arch() {
    log_info "Building multi-architecture Docker image: ${FULL_IMAGE_NAME}"
    log_info "Platforms: linux/amd64,linux/arm64"

    cd "${PROJECT_ROOT}"

    # Create/use buildx builder
    docker buildx create --name nestor-builder --use >/dev/null 2>&1 || true
    docker buildx use nestor-builder

    local push_flag=""
    if [ "${PUSH}" = "true" ]; then
        push_flag="--push"
        log_info "Will push to registry after build"
    else
        push_flag="--load"
        log_warning "Multi-arch build will only load amd64 image locally"
    fi

    docker buildx build \
        --file orchestrator/deployments/docker/Dockerfile \
        --tag "${FULL_IMAGE_NAME}" \
        --platform "linux/amd64,linux/arm64" \
        --build-arg VERSION="${VERSION}" \
        --build-arg COMMIT="${COMMIT}" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        ${push_flag} \
        .

    log_success "Multi-architecture image built: ${FULL_IMAGE_NAME}"
}

test_image() {
    log_info "Testing built image..."

    # Test image runs and shows version
    if docker run --rm "${FULL_IMAGE_NAME}" --version >/dev/null 2>&1; then
        log_success "Image test passed"
    else
        log_warning "Image test failed - but image was built"
    fi

    # Show image layers
    log_info "Image information:"
    docker inspect "${FULL_IMAGE_NAME}" --format='{{.Config.Labels}}' | tr ',' '\n' | sort
}

push_image() {
    if [ "${PUSH}" = "true" ]; then
        log_info "Pushing image to registry..."
        docker push "${FULL_IMAGE_NAME}"
        log_success "Image pushed: ${FULL_IMAGE_NAME}"
    fi
}

main() {
    local multi_arch=false

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--tag)
                IMAGE_TAG="$2"
                FULL_IMAGE_NAME="${IMAGE_NAME}:${IMAGE_TAG}"
                shift 2
                ;;
            -n|--name)
                IMAGE_NAME="$2"
                FULL_IMAGE_NAME="${IMAGE_NAME}:${IMAGE_TAG}"
                shift 2
                ;;
            --platform)
                PLATFORMS="$2"
                shift 2
                ;;
            --push)
                PUSH="true"
                shift
                ;;
            --multi-arch)
                multi_arch=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    # Show build information
    echo "🐳 Docker Build Configuration"
    echo "============================="
    echo "Image:       ${FULL_IMAGE_NAME}"
    echo "Version:     ${VERSION}"
    echo "Commit:      ${COMMIT}"
    echo "Build Date:  ${BUILD_DATE}"
    echo "Platforms:   ${PLATFORMS}"
    echo "Push:        ${PUSH}"
    echo "Multi-arch:  ${multi_arch}"
    echo ""

    # Build the image
    if [ "${multi_arch}" = "true" ]; then
        build_multi_arch
    else
        build_single_arch
        test_image
        push_image
    fi

    log_success "Build complete! 🎉"
    echo ""
    echo "🚀 Next steps:"
    echo "  Run locally:    docker run --rm -p 8080:8080 ${FULL_IMAGE_NAME}"
    echo "  With compose:   cd orchestrator/deployments/docker && docker-compose up"
    echo "  Deploy to k8s:  Use the Helm chart with image: ${FULL_IMAGE_NAME}"
}

# Run main function
main "$@"

---
# orchestrator/Makefile.local (add Docker targets)
# Additional Docker-related targets

.PHONY: docker-build docker-run docker-compose docker-push docker-test docker-clean

# Docker configuration
DOCKER_IMAGE := nestor/orchestrator
DOCKER_TAG := dev
DOCKER_FULL_NAME := $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-build: ## Build Docker image for orchestrator
	@echo "🐳 Building Docker image..."
	chmod +x deployments/scripts/docker-build.sh
	VERSION=$(DOCKER_TAG) ./deployments/scripts/docker-build.sh

docker-run: docker-build ## Run orchestrator in Docker (requires external deps)
	@echo "🚀 Running orchestrator in Docker..."
	docker run --rm -p 8080:8080 \
		-e LOG_LEVEL=debug \
		-e STORAGE_TYPE=memory \
		$(DOCKER_FULL_NAME)

docker-compose: ## Run full stack with docker-compose
	@echo "🐳 Starting full stack with docker-compose..."
	cd deployments/docker && docker-compose up --build

docker-compose-down: ## Stop docker-compose stack
	@echo "🛑 Stopping docker-compose stack..."
	cd deployments/docker && docker-compose down

docker-test: docker-build ## Test Docker image
	@echo "🧪 Testing Docker image..."
	docker run --rm $(DOCKER_FULL_NAME) --version
	docker run --rm $(DOCKER_FULL_NAME) --help

docker-push: docker-build ## Build and push Docker image
	@echo "📤 Pushing Docker image..."
	PUSH=true ./deployments/scripts/docker-build.sh

docker-multi-arch: ## Build multi-architecture image
	@echo "🏗️ Building multi-architecture image..."
	./deployments/scripts/docker-build.sh --multi-arch --push

docker-clean: ## Clean Docker images and containers
	@echo "🧹 Cleaning Docker resources..."
	docker container prune -f
	docker image prune -f
	docker rmi $(DOCKER_FULL_NAME) 2>/dev/null || true
