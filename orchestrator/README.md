# Nestor Orchestrator

The Orchestrator is the central coordination hub of the Nestor platform. It manages the resource catalog (the "buffet"), coordinates deployments across multiple engines, handles cross-team dependencies, and integrates with GitOps workflows.

## 🏗️ Architecture Overview

The Orchestrator follows a **plugin-based, interface-driven architecture** designed for scalability and extensibility. Core components:

- **Resource Catalog**: Git-sourced "buffet" of infrastructure components stored in DynamoDB
- **Deployment Coordination**: Routes deployments to appropriate engines (Crossplane, Pulumi, Terraform, Helm)
- **Dependency Management**: Maps and validates cross-team resource dependencies
- **GitOps Integration**: Commits manifests and coordinates with ArgoCD
- **Real-time Updates**: Server-Sent Events (SSE) for live processor notifications

## 📁 Directory Structure

```
orchestrator/
├── internal/                       # Private orchestrator implementation
│   ├── api/                        # HTTP API layer
│   ├── catalog/                    # Resource catalog management ("buffet")
│   ├── deployment/                 # Deployment engine coordination
│   ├── dependencies/               # Cross-team dependency handling
│   ├── gitops/                     # Git integration & ArgoCD
│   ├── events/sse/                 # Server-Sent Events for processors
│   ├── storage/                    # Storage layer (DynamoDB, cache)
│   ├── teams/                      # Team management & permissions
│   ├── policies/                   # Global policy enforcement
│   └── observability/              # Metrics, logging, tracing
├── pkg/                           # Public APIs for external consumption
│   ├── api/                       # Client libraries
│   ├── models/                    # Data models
│   └── events/                    # Event definitions
├── configs/                       # Environment configurations
├── deployments/                   # K8s/Helm/Docker deployment manifests
├── examples/                      # Sample resource definitions
└── docs/                         # Documentation
```

## 🚀 Development Roadmap

### **Phase 1: Foundation & Core Interfaces** 🎯
> **Goal**: Establish the interface-driven architecture and core data models

#### 1.1 Core Interfaces & Models (Week 1-2)
- [ ] **Storage interfaces** (`internal/storage/interface.go`)
  - CatalogStore, DeploymentStore, TeamStore interfaces
  - Cache interface for performance layer
- [ ] **Data models** (`pkg/models/`)
  - ResourceDefinition, DeploymentSpec, TeamConfig, Dependency
  - Event types for SSE communication
- [ ] **API types** (`pkg/api/types.go`)
  - Request/response structures for all endpoints
  - Error handling and status codes

#### 1.2 Configuration System (Week 2)
- [ ] **Config management** (`internal/config/`)
  - Environment-specific configuration loading
  - Validation and defaults
  - Hot-reload capability for development

#### 1.3 Basic HTTP Server (Week 2-3)
- [ ] **API foundation** (`internal/api/`)
  - HTTP server setup with middleware
  - Authentication, CORS, rate limiting
  - Health check endpoints
  - Request/response logging

### **Phase 2: Resource Catalog ("The Buffet")** 📚
> **Goal**: Implement the core resource catalog functionality

#### 2.1 DynamoDB Storage Implementation (Week 3-4)
- [ ] **DynamoDB client** (`internal/storage/dynamodb/`)
  - Table schemas and GSI design
  - CRUD operations for resource definitions
  - Team-based filtering and permissions
  - Migration system for schema changes

#### 2.2 Git Integration (Week 4-5)
- [ ] **Git client** (`internal/git/`)
  - Repository cloning and watching
  - Webhook handling for real-time updates
  - YAML parsing of resource definitions
  - Multi-provider support (GitHub, GitLab, Gitea)

#### 2.3 Catalog Management (Week 5-6)
- [ ] **Catalog manager** (`internal/catalog/`)
  - Git → DynamoDB synchronization
  - Resource definition validation
  - Versioning and change tracking
  - Team-based access control

#### 2.4 Catalog API Endpoints (Week 6)
- [ ] **Catalog handlers** (`internal/api/handlers/catalog.go`)
  - GET /catalog - List available resources for team
  - GET /catalog/{resource} - Get specific resource details
  - Team-based filtering and permissions
  - Pagination and search capabilities

### **Phase 3: Server-Sent Events System** ⚡
> **Goal**: Real-time communication with processors

#### 3.1 SSE Infrastructure (Week 7-8)
- [ ] **SSE server** (`internal/events/sse/`)
  - Client connection management
  - Event broadcasting with team filtering
  - Automatic reconnection handling
  - Connection cleanup and monitoring

#### 3.2 Event System (Week 8)
- [ ] **Event bus** (`internal/events/`)
  - Event generation from catalog changes
  - Event filtering and routing
  - Event persistence for offline processors
  - Event replay capabilities

#### 3.3 SSE API Endpoints (Week 8)
- [ ] **SSE handler** (`internal/api/handlers/sse.go`)
  - GET /events - SSE endpoint for processors
  - Team-based event filtering
  - Connection monitoring and health

### **Phase 4: Deployment Coordination** 🚀
> **Goal**: Multi-engine deployment orchestration

#### 4.1 Deployment Engine Interfaces (Week 9)
- [ ] **Engine interfaces** (`internal/deployment/engine/`)
  - DeploymentEngine interface definition
  - Engine registration and discovery
  - Engine health monitoring
  - Error handling and retry logic

#### 4.2 Engine Implementations (Week 10-12)
- [ ] **Crossplane engine** (`internal/deployment/engine/crossplane.go`)
  - Composition and XRD generation
  - Claim creation and management
  - Status monitoring and reporting
- [ ] **Pulumi engine** (`internal/deployment/engine/pulumi.go`)
  - Program generation and execution
  - Stack management per environment
  - State management and drift detection
- [ ] **Terraform engine** (`internal/deployment/engine/terraform.go`)
  - Configuration generation
  - Workspace management
  - State backend integration
- [ ] **Helm engine** (`internal/deployment/engine/helm.go`)
  - Chart generation and templating
  - Release management
  - Values.yaml per environment

#### 4.3 Deployment Coordination (Week 12-13)
- [ ] **Deployment coordinator** (`internal/deployment/coordinator.go`)
  - Engine selection and routing
  - Deployment queue management
  - Status tracking and reporting
  - Rollback capabilities

#### 4.4 Deployment API (Week 13)
- [ ] **Deployment handlers** (`internal/api/handlers/deployment.go`)
  - POST /deploy - Submit deployment request
  - GET /deployments - List deployments
  - GET /deployments/{id} - Get deployment status
  - DELETE /deployments/{id} - Cancel/rollback deployment

### **Phase 5: Dependency Management** 🔗
> **Goal**: Cross-team dependency mapping and validation

#### 5.1 Dependency Mapping (Week 14-15)
- [ ] **Dependency mapper** (`internal/dependencies/mapper.go`)
  - Dependency graph construction
  - Circular dependency detection
  - Cross-team dependency tracking
  - Impact analysis for changes

#### 5.2 Dependency Validation (Week 15)
- [ ] **Dependency validator** (`internal/dependencies/validator.go`)
  - Cross-team permission validation
  - Resource compatibility checking
  - Deployment order calculation
  - Conflict resolution strategies

#### 5.3 Dependency API (Week 15)
- [ ] **Dependency handlers** (`internal/api/handlers/dependencies.go`)
  - POST /validate-dependencies - Validate deployment dependencies
  - GET /dependencies - List team dependencies
  - GET /dependencies/graph - Visualize dependency graph

### **Phase 6: GitOps Integration** 📦
> **Goal**: Git repository management and ArgoCD integration

#### 6.1 Git Operations (Week 16-17)
- [ ] **Git manager** (`internal/gitops/manager.go`)
  - Multi-repository management
  - Branch strategies and management
  - Commit message templating
  - Merge conflict resolution

#### 6.2 ArgoCD Integration (Week 17-18)
- [ ] **ArgoCD client** (`internal/gitops/argocd.go`)
  - Application creation and management
  - Sync status monitoring
  - Health status tracking
  - Integration with deployment status

#### 6.3 GitOps API (Week 18)
- [ ] **GitOps handlers** (`internal/api/handlers/gitops.go`)
  - GET /gitops/status - Git and ArgoCD status
  - POST /gitops/sync - Trigger manual sync
  - GET /gitops/history - Deployment history

### **Phase 7: Team Management & Policies** 👥
> **Goal**: Multi-tenancy and policy enforcement

#### 7.1 Team Management (Week 19)
- [ ] **Team manager** (`internal/teams/manager.go`)
  - Team onboarding and configuration
  - Permission management
  - Quota tracking and enforcement
  - Team-specific metrics

#### 7.2 Policy Engine (Week 19-20)
- [ ] **Policy engine** (`internal/policies/engine.go`)
  - Global policy evaluation
  - Team-specific policy overrides
  - Violation reporting and handling
  - Policy as code from Git

#### 7.3 Team & Policy API (Week 20)
- [ ] **Team handlers** (`internal/api/handlers/teams.go`)
  - GET /teams - List teams and permissions
  - POST /teams/{id}/validate - Validate team deployment
  - GET /teams/{id}/quotas - Get team quota usage

### **Phase 8: Observability & Production Readiness** 📊
> **Goal**: Production monitoring and operational excellence

#### 8.1 Metrics & Monitoring (Week 21)
- [ ] **Metrics collection** (`internal/observability/metrics/`)
  - Prometheus metrics export
  - Custom business metrics
  - Performance monitoring
  - Resource utilization tracking

#### 8.2 Logging & Tracing (Week 21-22)
- [ ] **Structured logging** (`internal/observability/logging/`)
  - Context-aware logging
  - Log correlation across requests
  - Audit trail for compliance
- [ ] **Distributed tracing** (`internal/observability/tracing/`)
  - Jaeger integration
  - Request tracing across components
  - Performance bottleneck identification

#### 8.3 Health Checks & Diagnostics (Week 22)
- [ ] **Health monitoring** (`internal/observability/health/`)
  - Kubernetes readiness/liveness probes
  - Dependency health checking
  - Circuit breaker patterns
  - Graceful shutdown handling

### **Phase 9: Client Libraries & Documentation** 📖
> **Goal**: External integration and developer experience

#### 9.1 Client Libraries (Week 23)
- [ ] **Go client** (`pkg/api/client.go`)
  - Full API client for processors
  - SSE client for real-time updates
  - Retry logic and error handling
  - Connection pooling and optimization

#### 9.2 Documentation (Week 23-24)
- [ ] **API documentation** (OpenAPI/Swagger specs)
- [ ] **Deployment guides** (K8s, Docker, Helm)
- [ ] **Configuration reference** (All config options)
- [ ] **Troubleshooting guides** (Common issues and solutions)

#### 9.3 Examples & Templates (Week 24)
- [ ] **Resource definition examples** (`examples/resource-definitions/`)
- [ ] **Team configuration templates** (`examples/team-configs/`)
- [ ] **Policy examples** (`examples/policies/`)
- [ ] **Integration examples** (How to integrate with existing systems)

## 🎯 Success Criteria

### **Phase 1-3: Foundation (Weeks 1-8)**
- [ ] Processors can connect via SSE and receive catalog updates
- [ ] Resource definitions sync from Git to DynamoDB
- [ ] Basic API endpoints respond with proper authentication
- [ ] Health checks and basic metrics are operational

### **Phase 4-6: Core Functionality (Weeks 9-18)**
- [ ] End-to-end deployment through at least one engine (Crossplane)
- [ ] Cross-team dependencies are mapped and validated
- [ ] GitOps integration commits manifests to team repositories
- [ ] ArgoCD applications are created and monitored

### **Phase 7-9: Production Ready (Weeks 19-24)**
- [ ] Multi-team isolation and quota enforcement working
- [ ] Full observability stack (metrics, logs, traces) operational
- [ ] Complete documentation and examples available
- [ ] Production deployment successfully handling real workloads

## 🛠️ Development Principles

### **Interface-Driven Design**
- All external dependencies behind interfaces
- Plugin architecture for storage, deployment engines, git providers
- Easy to mock for testing, easy to extend for new providers

### **Event-Driven Architecture**
- SSE for real-time processor communication
- Event sourcing for audit trails
- Asynchronous processing where possible

### **Cloud-Native Best Practices**
- 12-factor app principles
- Health checks and graceful shutdown
- Structured logging and distributed tracing
- Configuration via environment variables

### **Scalability First**
- Horizontal scaling support from day one
- Database design optimized for read-heavy workloads
- Caching strategy for frequently accessed data
- Async processing for expensive operations

## 🚧 Getting Started

```bash
# Clone the repository
git clone https://github.com/nestor/nestor.git
cd nestor/orchestrator

# Install dependencies
go mod download

# Run tests
go test ./...

# Start development server
go run main.go serve --config configs/development.yaml
```

## 🤝 Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for development setup and contribution guidelines.

## 📋 Current Status

**🚧 In Development - Phase 1: Foundation**

- ✅ Directory structure and roadmap defined
- 🚧 Core interfaces and models in progress
- ⏳ DynamoDB storage implementation next
- ⏳ SSE system following storage layer
