#!/bin/bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
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

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Script directory and paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ORCHESTRATOR_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
HELM_CHART_DIR="$ORCHESTRATOR_ROOT/deployments/helm"
NAMESPACE="nestor-system"

# Configuration
MINIKUBE_PROFILE="nestor-dev"
ORCHESTRATOR_IMAGE="nestor/orchestrator:dev"

show_usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  setup     Setup Minikube cluster and deploy orchestrator"
    echo "  build     Build and load orchestrator image"
    echo "  deploy    Deploy orchestrator to existing cluster"
    echo "  status    Show deployment status"
    echo "  logs      Show orchestrator logs"
    echo "  shell     Open shell in orchestrator pod"
    echo "  forward   Start port forwarding"
    echo "  cleanup   Remove deployment"
    echo "  destroy   Destroy Minikube cluster"
    echo "  help      Show this help message"
}

check_dependencies() {
    log_info "Checking dependencies..."
    
    local missing_deps=()
    
    # Check required tools
    if ! command -v minikube >/dev/null 2>&1; then
        missing_deps+=("minikube")
    fi
    
    if ! command -v kubectl >/dev/null 2>&1; then
        missing_deps+=("kubectl")
    fi
    
    if ! command -v helm >/dev/null 2>&1; then
        missing_deps+=("helm")
    fi
    
    if ! command -v docker >/dev/null 2>&1; then
        missing_deps+=("docker")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        echo ""
        echo "Install instructions:"
        echo "  macOS: brew install minikube kubectl helm docker"
        echo "  Linux: Check your distribution's package manager"
        exit 1
    fi
    
    log_success "All dependencies found"
}

setup_minikube_cluster() {
    log_info "Setting up Minikube cluster..."
    
    # Check if profile already exists
    if minikube profile list -o json | jq -r '.valid[].Name' | grep -q "^${MINIKUBE_PROFILE}$" 2>/dev/null; then
        log_info "Minikube profile '${MINIKUBE_PROFILE}' already exists"
        
        # Check if it's running
        if minikube status -p "${MINIKUBE_PROFILE}" | grep -q "host: Running"; then
            log_info "Minikube cluster is already running"
        else
            log_info "Starting existing Minikube cluster..."
            minikube start -p "${MINIKUBE_PROFILE}"
        fi
    else
        log_info "Creating new Minikube cluster..."
        minikube start -p "${MINIKUBE_PROFILE}" \
            --cpus=4 \
            --memory=4096 \
            --disk-size=20g \
            --driver=docker \
            --kubernetes-version=v1.28.0
    fi
    
    # Set kubectl context
    kubectl config use-context "${MINIKUBE_PROFILE}"
    
    # Enable required addons
    log_info "Enabling Minikube addons..."
    minikube addons enable ingress -p "${MINIKUBE_PROFILE}"
    minikube addons enable metrics-server -p "${MINIKUBE_PROFILE}"
    
    log_success "Minikube cluster ready"
}

build_orchestrator_image() {
    log_info "Building orchestrator Docker image..."
    
    cd "$ORCHESTRATOR_ROOT"
    
    # Build the image
    docker build -t "${ORCHESTRATOR_IMAGE}" -f deployments/docker/Dockerfile .
    
    # Load image into Minikube
    log_info "Loading image into Minikube..."
    minikube image load "${ORCHESTRATOR_IMAGE}" -p "${MINIKUBE_PROFILE}"
    
    log_success "Orchestrator image built and loaded"
}

deploy_dependencies() {
    log_info "Deploying dependencies..."
    
    # Create namespace
    kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -
    
    # Add Bitnami Helm repo for Redis
    helm repo add bitnami https://charts.bitnami.com/bitnami >/dev/null 2>&1 || true
    helm repo update >/dev/null
    
    # Deploy Redis
    log_info "Deploying Redis..."
    helm upgrade --install nestor-redis bitnami/redis \
        --namespace "${NAMESPACE}" \
        --set auth.enabled=false \
        --set master.persistence.enabled=false \
        --set replica.replicaCount=0 \
        --set master.resources.requests.cpu=50m \
        --set master.resources.requests.memory=64Mi \
        --set master.resources.limits.cpu=200m \
        --set master.resources.limits.memory=128Mi \
        --wait --timeout=300s
    
    # Deploy DynamoDB Local
    log_info "Deploying DynamoDB Local..."
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nestor-dynamodb-local
  namespace: ${NAMESPACE}
  labels:
    app: nestor-dynamodb-local
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nestor-dynamodb-local
  template:
    metadata:
      labels:
        app: nestor-dynamodb-local
    spec:
      containers:
        - name: dynamodb-local
          image: amazon/dynamodb-local:2.0.0
          command: ["-jar", "DynamoDBLocal.jar", "-sharedDb", "-optimizeDbBeforeStartup"]
          ports:
            - containerPort: 8000
          resources:
            requests:
              cpu: 50m
              memory: 128Mi
            limits:
              cpu: 200m
              memory: 256Mi
          readinessProbe:
            httpGet:
              path: /
              port: 8000
            initialDelaySeconds: 10
            periodSeconds: 5
          livenessProbe:
            httpGet:
              path: /
              port: 8000
            initialDelaySeconds: 30
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: nestor-dynamodb-local
  namespace: ${NAMESPACE}
  labels:
    app: nestor-dynamodb-local
spec:
  ports:
    - port: 8000
      targetPort: 8000
      name: http
  selector:
    app: nestor-dynamodb-local
EOF
    
    # Wait for DynamoDB to be ready
    log_info "Waiting for DynamoDB Local to be ready..."
    kubectl wait --for=condition=available --timeout=300s deployment/nestor-dynamodb-local -n "${NAMESPACE}"
    
    log_success "Dependencies deployed"
}

deploy_orchestrator() {
    log_info "Deploying Nestor Orchestrator..."
    
    # Create values file for Minikube
    cat > "${HELM_CHART_DIR}/values-minikube.yaml" << EOF
replicaCount: 1

image:
  repository: nestor/orchestrator
  tag: dev
  pullPolicy: Never  # Use local image

resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi

config:
  logging:
    level: "debug"
    format: "console"

  storage:
    dynamodb:
      endpoint: "http://nestor-dynamodb-local:8000"
      tableName: "nestor-components-dev"
      readCapacity: 2
      writeCapacity: 2

  cache:
    redis:
      url: "redis://nestor-redis-master:6379"

  git:
    repositories:
      - name: "local-components"
        url: "https://github.com/HatiCode/nestor.git"
        path: "examples/components"
        branch: "main"
        pollInterval: "30s"

  sse:
    maxConnections: 50

# Don't deploy dependencies - we handle them separately
dependencies:
  redis:
    enabled: false
  dynamodbLocal:
    enabled: false

# Service configuration for easy access
service:
  type: NodePort
  port: 80
  targetPort: 8080

# Enable ingress for easy access
ingress:
  enabled: true
  className: "nginx"
  hosts:
    - host: orchestrator.nestor.local
      paths:
        - path: /
          pathType: Prefix
EOF
    
    # Deploy using Helm
    helm upgrade --install nestor-orchestrator "${HELM_CHART_DIR}" \
        --namespace "${NAMESPACE}" \
        --values "${HELM_CHART_DIR}/values-minikube.yaml" \
        --wait --timeout=300s
    
    log_success "Orchestrator deployed"
}

show_access_info() {
    log_info "Getting access information..."
    
    # Get Minikube IP
    MINIKUBE_IP=$(minikube ip -p "${MINIKUBE_PROFILE}")
    
    # Get NodePort
    NODEPORT=$(kubectl get service nestor-orchestrator -n "${NAMESPACE}" -o jsonpath='{.spec.ports[0].nodePort}')
    
    # Get ingress info
    INGRESS_ENABLED=$(kubectl get ingress nestor-orchestrator -n "${NAMESPACE}" 2>/dev/null && echo "true" || echo "false")
    
    echo ""
    echo -e "${CYAN}🎉 Nestor Orchestrator is now running!${NC}"
    echo "=================================="
    echo ""
    echo "📋 Access Information:"
    echo "  Direct URL:    http://${MINIKUBE_IP}:${NODEPORT}"
    
    if [ "$INGRESS_ENABLED" = "true" ]; then
        echo "  Ingress URL:   http://orchestrator.nestor.local"
        echo ""
        echo "📝 Add to /etc/hosts for ingress access:"
        echo "  echo '${MINIKUBE_IP} orchestrator.nestor.local' | sudo tee -a /etc/hosts"
    fi
    
    echo ""
    echo "🔗 Useful URLs:"
    echo "  Health:        http://${MINIKUBE_IP}:${NODEPORT}/health"
    echo "  Ready:         http://${MINIKUBE_IP}:${NODEPORT}/ready"
    echo "  Metrics:       http://${MINIKUBE_IP}:${NODEPORT}/metrics"
    echo "  Components:    http://${MINIKUBE_IP}:${NODEPORT}/api/v1/components"
    echo ""
    echo "🛠️  Management Commands:"
    echo "  Status:        $0 status"
    echo "  Logs:          $0 logs"
    echo "  Shell:         $0 shell"
    echo "  Port Forward:  $0 forward"
    echo ""
}

show_status() {
    log_info "Checking deployment status..."
    
    echo ""
    echo "📋 Minikube Status:"
    echo "==================="
    minikube status -p "${MINIKUBE_PROFILE}" || true
    
    echo ""
    echo "📋 Kubernetes Resources:"
    echo "========================"
    kubectl get pods,services,ingress -n "${NAMESPACE}"
    
    echo ""
    echo "📋 Orchestrator Pod Details:"
    echo "============================="
    kubectl describe pod -l app.kubernetes.io/name=nestor-orchestrator -n "${NAMESPACE}" | head -20
    
    echo ""
    echo "📋 Recent Events:"
    echo "================="
    kubectl get events -n "${NAMESPACE}" --sort-by='.metadata.creationTimestamp' | tail -10
}

show_logs() {
    log_info "Showing orchestrator logs..."
    kubectl logs -f -l app.kubernetes.io/name=nestor-orchestrator -n "${NAMESPACE}"
}

open_shell() {
    log_info "Opening shell in orchestrator pod..."
    local pod_name
    pod_name=$(kubectl get pods -l app.kubernetes.io/name=nestor-orchestrator -n "${NAMESPACE}" -o jsonpath='{.items[0].metadata.name}')
    
    if [ -z "$pod_name" ]; then
        log_error "No orchestrator pod found"
        exit 1
    fi
    
    kubectl exec -it "$pod_name" -n "${NAMESPACE}" -- /bin/sh
}

port_forward() {
    log_info "Starting port forwarding..."
    local pod_name
    pod_name=$(kubectl get pods -l app.kubernetes.io/name=nestor-orchestrator -n "${NAMESPACE}" -o jsonpath='{.items[0].metadata.name}')
    
    if [ -z "$pod_name" ]; then
        log_error "No orchestrator pod found"
        exit 1
    fi
    
    log_success "Orchestrator will be available at: http://localhost:8080"
    log_info "Press Ctrl+C to stop port forwarding"
    kubectl port-forward "$pod_name" 8080:8080 -n "${NAMESPACE}"
}

cleanup_deployment() {
    log_info "Cleaning up deployment..."
    
    # Remove Helm releases
    helm uninstall nestor-orchestrator -n "${NAMESPACE}" >/dev/null 2>&1 || true
    helm uninstall nestor-redis -n "${NAMESPACE}" >/dev/null 2>&1 || true
    
    # Remove DynamoDB Local
    kubectl delete deployment,service nestor-dynamodb-local -n "${NAMESPACE}" >/dev/null 2>&1 || true
    
    # Remove values file
    rm -f "${HELM_CHART_DIR}/values-minikube.yaml"
    
    log_success "Deployment cleaned up"
}

destroy_cluster() {
    log_info "Destroying Minikube cluster..."
    
    # Stop and delete the cluster
    minikube delete -p "${MINIKUBE_PROFILE}"
    
    log_success "Minikube cluster destroyed"
}

test_deployment() {
    log_info "Testing deployment..."
    
    # Get service endpoint
    MINIKUBE_IP=$(minikube ip -p "${MINIKUBE_PROFILE}")
    NODEPORT=$(kubectl get service nestor-orchestrator -n "${NAMESPACE}" -o jsonpath='{.spec.ports[0].nodePort}')
    BASE_URL="http://${MINIKUBE_IP}:${NODEPORT}"
    
    # Test health endpoint
    if curl -s -f "${BASE_URL}/health" >/dev/null; then
        log_success "Health check passed"
    else
        log_error "Health check failed"
        return 1
    fi
    
    # Test ready endpoint
    if curl -s -f "${BASE_URL}/ready" >/dev/null; then
        log_success "Ready check passed"
    else
        log_warning "Ready check failed (might still be starting)"
    fi
    
    # Test metrics endpoint
    if curl -s -f "${BASE_URL}/metrics" >/dev/null; then
        log_success "Metrics endpoint accessible"
    else
        log_warning "Metrics endpoint not accessible"
    fi
    
    log_success "Deployment test completed"
}

main() {
    local command="${1:-setup}"
    
    case "$command" in
        "setup")
            log_info "🚀 Setting up complete Minikube environment..."
            check_dependencies
            setup_minikube_cluster
            build_orchestrator_image
            deploy_dependencies
            deploy_orchestrator
            
            # Wait a bit for everything to settle
            sleep 10
            test_deployment
            show_access_info
            
            log_success "🎉 Setup complete!"
            ;;
        "build")
            check_dependencies
            build_orchestrator_image
            ;;
        "deploy")
            check_dependencies
            deploy_dependencies
            deploy_orchestrator
            show_access_info
            ;;
        "status")
            show_status
            ;;
        "logs")
            show_logs
            ;;
        "shell")
            open_shell
            ;;
        "forward")
            port_forward
            ;;
        "test")
            test_deployment
            ;;
        "cleanup")
            cleanup_deployment
            ;;
        "destroy")
            cleanup_deployment
            destroy_cluster
            ;;
        "help"|"-h"|"--help")
            show_usage
            ;;
        *)
            log_error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

# Handle script interruption
trap 'log_error "Script interrupted"; exit 1' INT TERM

# Run main function
main "$@"

---
# orchestrator/deployments/scripts/minikube-test.sh
#!/bin/bash
set -euo pipefail

# Simple test script for the Minikube deployment

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MINIKUBE_PROFILE="nestor-dev"
NAMESPACE="nestor-system"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "Testing $test_name... "
    if eval "$test_command" >/dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        return 1
    fi
}

main() {
    echo "🧪 Testing Nestor Orchestrator deployment on Minikube"
    echo "====================================================="
    
    # Get service details
    MINIKUBE_IP=$(minikube ip -p "${MINIKUBE_PROFILE}")
    NODEPORT=$(kubectl get service nestor-orchestrator -n "${NAMESPACE}" -o jsonpath='{.spec.ports[0].nodePort}' 2>/dev/null || echo "")
    
    if [ -z "$NODEPORT" ]; then
        echo -e "${RED}❌ Could not find orchestrator service${NC}"
        exit 1
    fi
    
    BASE_URL="http://${MINIKUBE_IP}:${NODEPORT}"
    
    echo "🔗 Testing URL: $BASE_URL"
    echo ""
    
    # Run tests
    local failed=0
    
    run_test "Minikube cluster" "minikube status -p ${MINIKUBE_PROFILE} | grep -q 'host: Running'" || ((failed++))
    run_test "Namespace exists" "kubectl get namespace ${NAMESPACE}" || ((failed++))
    run_test "Orchestrator pod running" "kubectl get pods -l app.kubernetes.io/name=nestor-orchestrator -n ${NAMESPACE} | grep -q Running" || ((failed++))
    run_test "Redis pod running" "kubectl get pods -l app.kubernetes.io/name=redis -n ${NAMESPACE} | grep -q Running" || ((failed++))
    run_test "DynamoDB pod running" "kubectl get pods -l app=nestor-dynamodb-local -n ${NAMESPACE} | grep -q Running" || ((failed++))
    run_test "Health endpoint" "curl -s -f ${BASE_URL}/health" || ((failed++))
    run_test "Ready endpoint" "curl -s -f ${BASE_URL}/ready" || ((failed++))
    run_test "Metrics endpoint" "curl -s -f ${BASE_URL}/metrics" || ((failed++))
    
    echo ""
    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}✅ All tests passed!${NC}"
        echo ""
        echo "🎯 Try these URLs:"
        echo "   Health: $BASE_URL/health"
        echo "   API:    $BASE_URL/api/v1/components"
        echo ""
        echo "🛠️  Useful commands:"
        echo "   Logs:   kubectl logs -f -l app.kubernetes.io/name=nestor-orchestrator -n ${NAMESPACE}"
        echo "   Shell:  kubectl exec -it \$(kubectl get pods -l app.kubernetes.io/name=nestor-orchestrator -n ${NAMESPACE} -o name) -n ${NAMESPACE} -- /bin/sh"
        echo "   Port:   kubectl port-forward svc/nestor-orchestrator 8080:80 -n ${NAMESPACE}"
        exit 0
    else
        echo -e "${RED}❌ $failed test(s) failed${NC}"
        echo ""
        echo "🔍 Troubleshooting:"
        echo "   Check status: $SCRIPT_DIR/minikube-setup.sh status"
        echo "   View logs:    $SCRIPT_DIR/minikube-setup.sh logs"
        exit 1
    fi
}

main "$@"

---
# orchestrator/Makefile.local (updated)
# Orchestrator-specific Makefile for Minikube

.PHONY: minikube-setup minikube-build minikube-deploy minikube-status minikube-logs minikube-shell minikube-forward minikube-test minikube-cleanup minikube-destroy

# Minikube development targets
MINIKUBE_SCRIPT := deployments/scripts/minikube-setup.sh
TEST_SCRIPT := deployments/scripts/minikube-test.sh

minikube-setup: ## Setup complete Minikube environment with orchestrator
	@echo "🚀 Setting up Minikube environment..."
	chmod +x $(MINIKUBE_SCRIPT)
	$(MINIKUBE_SCRIPT) setup

minikube-build: ## Build and load orchestrator image into Minikube
	@echo "🐳 Building orchestrator image for Minikube..."
	chmod +x $(MINIKUBE_SCRIPT)
	$(MINIKUBE_SCRIPT) build

minikube-deploy: ## Deploy orchestrator to existing Minikube cluster
	@echo "📦 Deploying orchestrator to Minikube..."
	chmod +x $(MINIKUBE_SCRIPT)
	$(MINIKUBE_SCRIPT) deploy

minikube-status: ## Show Minikube deployment status
	@echo "📋 Checking Minikube deployment status..."
	chmod +x $(MINIKUBE_SCRIPT)
	$(MINIKUBE_SCRIPT) status

minikube-logs: ## Show orchestrator logs in Minikube
	@echo "📜 Showing orchestrator logs..."
	chmod +x $(MINIKUBE_SCRIPT)
	$(MINIKUBE_SCRIPT) logs

minikube-shell: ## Open shell in orchestrator pod
	@echo "🐚 Opening shell in orchestrator pod..."
	chmod +x $(MINIKUBE_SCRIPT)
	$(MINIKUBE_SCRIPT) shell

minikube-forward: ## Start port forwarding to orchestrator
	@echo "🔗 Starting port forwarding..."
	chmod +x $(MINIKUBE_SCRIPT)
	$(MINIKUBE_SCRIPT) forward

minikube-test: ## Run tests against Minikube deployment
	@echo "🧪 Testing Minikube deployment..."
	chmod +x $(TEST_SCRIPT)
	$(TEST_SCRIPT)

minikube-cleanup: ## Remove orchestrator deployment from Minikube
	@echo "🧹 Cleaning up Minikube deployment..."
	chmod +x $(MINIKUBE_SCRIPT)
	$(MINIKUBE_SCRIPT) cleanup

minikube-destroy: ## Destroy Minikube cluster completely
	@echo "💥 Destroying Minikube cluster..."
	chmod +x $(MINIKUBE_SCRIPT)
	$(MINIKUBE_SCRIPT) destroy

# Aliases for convenience
dev-setup: minikube-setup ## Alias for minikube-setup
dev-status: minikube-status ## Alias for minikube-status
dev-logs: minikube-logs ## Alias for minikube-logs
dev-clean: minikube-cleanup ## Alias for minikube-cleanup