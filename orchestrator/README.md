# Nestor Orchestrator

The Orchestrator is the **deployment coordination engine** of the Nestor platform. It receives deployment requests from Composers, resolves complex dependencies, and coordinates deployments across multiple infrastructure engines (Crossplane, Pulumi, Terraform, Helm).

## 🏗️ Architecture Overview

The Orchestrator is designed as the **middle layer** between Composers (team abstractions) and the Catalog (infrastructure primitives):

```
Composers → Orchestrator → Catalog
(Deploy)   (Coordinate)   (Resources)
```

**Key Responsibilities:**
- **Deployment Coordination**: Routes deployments to appropriate engines
- **Dependency Resolution**: Manages complex resource dependencies
- **Multi-Engine Support**: Coordinates Crossplane, Pulumi, Terraform, Helm
- **GitOps Integration**: Commits manifests and coordinates with ArgoCD
- **Status Tracking**: Monitors deployment progress and health

## 📁 Directory Structure

```
orchestrator/
├── main.go                           # Application entry point
├── internal/                         # Private orchestrator implementation
│   ├── api/                          # HTTP API layer
│   │   ├── handlers/                 # HTTP request handlers
│   │   │   ├── deployments.go        # Deployment orchestration endpoints
│   │   │   ├── engines.go            # Engine management endpoints
│   │   │   ├── health.go             # Health check endpoints
│   │   │   └── status.go             # Deployment status endpoints
│   │   └── middleware/               # HTTP middleware
│   │
│   ├── deployment/                   # Deployment coordination
│   │   ├── coordinator.go            # Main deployment coordinator
│   │   ├── dependency.go             # Dependency resolution logic
│   │   ├── queue.go                  # Deployment queue management
│   │   └── status.go                 # Status tracking and reporting
│   │
│   ├── engines/                      # Deployment engine management
│   │   ├── interfaces.go             # Engine interfaces
│   │   ├── registry.go               # Engine registry implementation
│   │   ├── crossplane/               # Crossplane engine
│   │   ├── pulumi/                   # Pulumi engine
│   │   ├── terraform/                # Terraform engine
│   │   └── helm/                     # Helm engine
│   │
│   ├── catalog/                      # Catalog service client
│   │   ├── client.go                 # HTTP client for catalog service
│   │   └── cache.go                  # Local cache for catalog data
│   │
│   ├── gitops/                       # GitOps integration
│   │   ├── manager.go                # Git repository management
│   │   ├── argocd.go                 # ArgoCD integration
│   │   └── manifest.go               # Manifest generation
│   │
│   └── observability/                # Metrics, logging, tracing
│       ├── metrics/                  # Prometheus metrics
│       ├── logging/                  # Structured logging
│       └── tracing/                  # Distributed tracing
│
├── pkg/                              # Public APIs
│   ├── api/                          # Client libraries
│   │   ├── client.go                 # Orchestrator API client
│   │   └── types.go                  # Request/response types
│   └── models/                       # Data models
│       ├── deployment.go             # Deployment models
│       ├── engine.go                 # Engine models
│       └── status.go                 # Status models
│
├── configs/                          # Configuration files
├── deployments/                      # K8s/Helm/Docker deployment manifests
└── docs/                            # Architecture documentation
```

## 🚀 Development Roadmap

### **Phase 1: Core Deployment Coordination** 🎯
> **Goal**: Establish deployment orchestration with dependency resolution

#### 1.1 Deployment Coordination (Week 1-2)
- [ ] **Deployment API** (`internal/api/handlers/deployments.go`)
  - POST /deployments - Accept deployment requests from composers
  - GET /deployments/{id} - Get deployment status
  - DELETE /deployments/{id} - Cancel/rollback deployment
- [ ] **Deployment Queue** (`internal/deployment/queue.go`)
  - Async deployment processing
  - Priority-based scheduling
  - Retry logic and backoff
- [ ] **Status Tracking** (`internal/deployment/status.go`)
  - Real-time deployment status
  - Progress reporting
  - Error handling and recovery

#### 1.2 Dependency Resolution (Week 2-3)
- [ ] **Dependency Mapper** (`internal/deployment/dependency.go`)
  - Resource dependency graph construction
  - Circular dependency detection
  - Deployment order calculation
- [ ] **Dependency Validator**
  - Cross-resource compatibility checking
  - Conflict resolution strategies
  - Impact analysis for changes

### **Phase 2: Engine Integration** 🔧
> **Goal**: Multi-engine deployment support

#### 2.1 Engine Framework (Week 3-4)
- [ ] **Engine Interfaces** (`internal/engines/interfaces.go`)
  - DeploymentEngine interface definition
  - Engine registration and discovery
  - Health monitoring and status reporting
- [ ] **Engine Registry** (`internal/engines/registry.go`)
  - Dynamic engine registration
  - Engine selection logic
  - Load balancing and failover

#### 2.2 Engine Implementations (Week 4-8)
- [ ] **Crossplane Engine** (`internal/engines/crossplane/`)
  - Composition and XRD generation
  - Claim creation and management
  - Status monitoring and reporting
- [ ] **Pulumi Engine** (`internal/engines/pulumi/`)
  - Program generation and execution
  - Stack management per environment
  - State management and drift detection
- [ ] **Terraform Engine** (`internal/engines/terraform/`)
  - Configuration generation
  - Workspace management
  - State backend integration
- [ ] **Helm Engine** (`internal/engines/helm/`)
  - Chart generation and templating
  - Release management
  - Values.yaml per environment

### **Phase 3: Catalog Integration** 📚
> **Goal**: Seamless integration with catalog service

#### 3.1 Catalog Client (Week 8-9)
- [ ] **HTTP Client** (`internal/catalog/client.go`)
  - Resource definition retrieval
  - Version resolution
  - Bulk operations support
- [ ] **Local Cache** (`internal/catalog/cache.go`)
  - Redis-backed resource caching
  - Cache invalidation strategies
  - Offline operation support

### **Phase 4: GitOps Integration** 📦
> **Goal**: Git repository management and ArgoCD coordination

#### 4.1 Git Operations (Week 9-10)
- [ ] **Git Manager** (`internal/gitops/manager.go`)
  - Multi-repository management
  - Branch strategies and management
  - Commit message templating
- [ ] **Manifest Generation** (`internal/gitops/manifest.go`)
  - Engine-specific manifest generation
  - Template rendering and validation
  - Environment-specific configurations

#### 4.2 ArgoCD Integration (Week 10-11)
- [ ] **ArgoCD Client** (`internal/gitops/argocd.go`)
  - Application creation and management
  - Sync status monitoring
  - Health status tracking

### **Phase 5: Advanced Features** 🚀
> **Goal**: Production-ready orchestration capabilities

#### 5.1 Advanced Deployment Patterns (Week 11-12)
- [ ] **Blue-Green Deployments**
- [ ] **Canary Deployments**
- [ ] **Rolling Updates**
- [ ] **Rollback Coordination**

#### 5.2 Observability & Monitoring (Week 12-13)
- [ ] **Metrics Collection** (`internal/observability/metrics/`)
  - Deployment success/failure rates
  - Engine performance metrics
  - Resource utilization tracking
- [ ] **Distributed Tracing** (`internal/observability/tracing/`)
  - End-to-end request tracing
  - Cross-service correlation
  - Performance bottleneck identification

## 🎯 API Design

### Deployment Request Flow
```
Composer → POST /deployments → Orchestrator
                              ↓
                        Dependency Resolution
                              ↓
                        Engine Selection
                              ↓
                        Deployment Execution
                              ↓
                        Status Updates
```

### Key API Endpoints

#### **Create Deployment**
```http
POST /api/v1/deployments
{
  "id": "team-alpha-web-app-v1",
  "composer": "team-alpha",
  "resources": [
    {
      "name": "database",
      "catalogRef": "aws-rds-mysql:1.0.0",
      "config": {...},
      "engine": "crossplane"
    },
    {
      "name": "deployment",
      "catalogRef": "k8s-deployment:2.1.0",
      "config": {...},
      "engine": "helm"
    }
  ],
  "dependencies": [
    {"from": "database", "to": "deployment"}
  ],
  "environment": "production"
}
```

#### **Get Deployment Status**
```http
GET /api/v1/deployments/team-alpha-web-app-v1
{
  "id": "team-alpha-web-app-v1",
  "status": "in_progress",
  "phase": "deploying",
  "resources": [
    {
      "name": "database",
      "status": "ready",
      "engine": "crossplane",
      "outputs": {"endpoint": "..."}
    },
    {
      "name": "deployment",
      "status": "in_progress",
      "engine": "helm",
      "progress": "50%"
    }
  ]
}
```

## 🔧 Configuration

### Engine Configuration
```yaml
engines:
  crossplane:
    enabled: true
    endpoint: "https://crossplane.platform.svc.cluster.local"
    timeout: "300s"

  pulumi:
    enabled: true
    backend: "s3://pulumi-state-bucket"
    parallelism: 10

  terraform:
    enabled: false
    backend: "remote"
    workspace_prefix: "nestor-"
```

### Catalog Integration
```yaml
catalog:
  endpoint: "https://catalog.nestor.svc.cluster.local"
  cache:
    enabled: true
    ttl: "5m"
    redis_url: "redis://redis.nestor.svc.cluster.local:6379"
```

### GitOps Settings
```yaml
gitops:
  enabled: true
  repositories:
    - name: "team-manifests"
      url: "https://github.com/company/team-manifests.git"
      branch: "main"
      path: "manifests/"

  argocd:
    endpoint: "https://argocd.platform.svc.cluster.local"
    namespace: "argocd"
```

## 🚀 Getting Started

### Local Development
```bash
# Start development environment
make dev-orchestrator

# With dependencies
make docker-up
make dev-orchestrator
```

### Docker Deployment
```bash
# Build and run
make docker-build
docker run -p 8080:8080 nestor/orchestrator:dev
```

### Kubernetes Deployment
```bash
# Using Helm
helm install nestor-orchestrator deployments/helm \
  --set config.catalog.endpoint=https://catalog.nestor.svc.cluster.local
```

## 🎯 Key Design Decisions

### **Stateless by Design**
- Orchestrator is stateless - all state in external stores
- Enables horizontal scaling and high availability
- Deployment state stored in engines and status services

### **Engine Abstraction**
- Pluggable engine architecture
- Each engine implements the same interface
- Easy to add new engines without changing core logic

### **Async Processing**
- All deployments are asynchronous
- Status updates via polling or webhooks
- Queue-based processing for reliability

### **Catalog Independence**
- Orchestrator can function with cached catalog data
- Graceful degradation if catalog is unavailable
- Eventually consistent resource definitions

## 🚨 Error Handling

### **Deployment Failures**
- Automatic rollback capabilities
- Partial deployment recovery
- Detailed error reporting and troubleshooting

### **Engine Failures**
- Circuit breaker patterns
- Failover to alternative engines
- Engine health monitoring

### **Dependency Issues**
- Validation before deployment
- Clear error messages for circular dependencies
- Impact analysis for failing dependencies

## 📊 Monitoring & Observability

### **Key Metrics**
- Deployment success/failure rates
- Average deployment time
- Engine performance and availability
- Queue depth and processing time

### **Logging**
- Structured logging with correlation IDs
- Deployment audit trails
- Engine operation logs

### **Tracing**
- End-to-end deployment tracing
- Cross-service request correlation
- Performance bottleneck identification

## 🤝 Integration Patterns

### **With Composers**
- REST API for deployment requests
- Async status updates
- Error propagation and handling

### **With Catalog**
- Resource definition retrieval
- Version resolution
- Caching for performance

### **With Engines**
- Pluggable interface pattern
- Health monitoring
- Status aggregation

---

**The Orchestrator is the brain of Nestor's deployment coordination - handling the complex orchestration so that Composers can focus on team abstractions and the Catalog can focus on resource definitions.**
