# Nestor Platform - Architecture & Development Guide

This document defines the architectural decisions, coding philosophy, and system design for the Nestor platform. **Read this first** before making any code changes to ensure consistency with the established patterns.

## 🏗️ System Architecture Overview

Nestor provides a complete platform engineering solution through three core services that work together to enable self-service infrastructure while maintaining platform team control.

### Core Design Principles

1. **Service Separation** - Each service has a single, well-defined responsibility
2. **Team Autonomy** - Product teams can create abstractions without platform team bottlenecks
3. **Platform Control** - Infrastructure teams maintain governance and standards
4. **Multi-Engine Support** - Not locked into any single IaC tool
5. **API-Driven** - All interactions happen through well-defined APIs
6. **Event-Driven** - Real-time updates and coordination between services

## 📊 Three-Layer Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                       🖥️  CLI Tool                              │
│                    Developer Interface                          │
└─────────────────────────┬───────────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────────┐
│                   🎵 COMPOSER SERVICE                           │
│              Team-Specific Resource Composition                 │
│                                                                 │
│  • Team abstractions (web-app, data-pipeline)                  │
│  • Business logic validation                                   │
│  • API exposure for teams                                      │
│  • Deployment request coordination                             │
└─────────────────────────┬───────────────────────────────────────┘
                          │ Deployment Requests
┌─────────────────────────▼───────────────────────────────────────┐
│                  🎼 ORCHESTRATOR SERVICE                        │
│              Deployment Engine & Coordination                   │
│                                                                 │
│  • Multi-engine deployment (Crossplane, Pulumi, Terraform)     │
│  • Complex dependency resolution                               │
│  • GitOps workflow management                                  │
│  • Status tracking and rollback                               │
└─────────────────────────┬───────────────────────────────────────┘
                          │ Resource Lookups
┌─────────────────────────▼───────────────────────────────────────┐
│                   📚 CATALOG SERVICE                            │
│              Infrastructure Resource Definitions                │
│                                                                 │
│  • Low-level resource primitives (RDS, S3, VPC)               │
│  • Versioning and validation                                  │
│  • Real-time updates via SSE                                  │
│  • Platform team governance                                   │
└─────────────────────────────────────────────────────────────────┘
```

## 🎯 Service Responsibilities

### 📚 **Catalog Service** - The Foundation
**Single Responsibility**: Manage the global infrastructure resource catalog

**Core Functions:**
- **Resource Definition Storage**: DynamoDB-backed storage for infrastructure primitives
- **Git Synchronization**: Real-time sync from platform team repositories
- **Version Management**: Semantic versioning for all resource definitions
- **Real-time Updates**: Server-Sent Events for live catalog changes
- **Validation & Governance**: Platform team-controlled resource schemas

**Technology Stack:**
- **Language**: Go
- **Storage**: DynamoDB (primary), Redis (cache)
- **Events**: Server-Sent Events (SSE)
- **Git Integration**: Multi-repository support with webhooks

**API Patterns:**
```
GET    /api/v1/resources                    # List available resources
GET    /api/v1/resources/{name}             # Get specific resource
GET    /api/v1/resources/{name}/versions    # List resource versions
GET    /api/v1/events                       # SSE stream for updates
```

### 🎼 **Orchestrator Service** - The Engine
**Single Responsibility**: Coordinate complex deployments across multiple engines

**Core Functions:**
- **Deployment Coordination**: Route deployments to appropriate engines
- **Dependency Resolution**: Manage complex resource dependencies
- **Multi-Engine Support**: Crossplane, Pulumi, Terraform, Helm coordination
- **GitOps Integration**: Manifest generation and ArgoCD coordination
- **Status Tracking**: Real-time deployment monitoring and rollback capabilities

**Technology Stack:**
- **Language**: Go
- **Engines**: Crossplane, Pulumi, Terraform, Helm
- **GitOps**: ArgoCD integration
- **Queue**: Redis-based async processing
- **State**: External engine state management

**API Patterns:**
```
POST   /api/v1/deployments                  # Create deployment
GET    /api/v1/deployments/{id}             # Get deployment status
DELETE /api/v1/deployments/{id}             # Cancel/rollback
GET    /api/v1/engines                      # List available engines
```

### 🎵 **Composer Service** - The Abstraction Layer
**Single Responsibility**: Enable teams to create business-focused resource abstractions

**Core Functions:**
- **Resource Composition**: Combine catalog primitives into team abstractions
- **Team API Exposure**: Provide team-specific APIs for infrastructure
- **Business Logic**: Team-specific validation and workflows
- **Deployment Coordination**: Interface with orchestrator for deployments

**Technology Stack:**
- **Language**: Go
- **Storage**: PostgreSQL (team compositions), Redis (cache)
- **Team Isolation**: Multi-tenant with namespace isolation
- **API Gateway**: Team-specific API endpoints

**API Patterns:**
```
GET    /api/v1/compositions                 # List team compositions
POST   /api/v1/compositions                 # Create new composition
POST   /api/v1/deploy                       # Deploy composed resources
GET    /api/v1/status/{deployment}          # Get deployment status
```

## 🔄 Data Flow & Interactions

### **Deployment Flow**
```
1. Developer updates code with annotations
   └─ //nestor:web-app size=large replicas=5

2. CLI parses annotations and calls Composer
   └─ POST /api/v1/deploy {composition: "web-app", params: {...}}

3. Composer resolves composition to catalog resources
   └─ GET /catalog/api/v1/resources/aws-rds-mysql:1.2.0

4. Composer sends deployment request to Orchestrator
   └─ POST /orchestrator/api/v1/deployments {...}

5. Orchestrator resolves dependencies and selects engines
   └─ database (crossplane) → deployment (helm) → load-balancer (terraform)

6. Orchestrator coordinates deployment across engines
   └─ Creates Crossplane XR → Waits for ready → Creates Helm release

7. Status updates flow back through the chain
   └─ Engine → Orchestrator → Composer → CLI → Developer
```

### **Catalog Update Flow**
```
1. Platform team commits new resource definition
   └─ git push origin main

2. Catalog service receives webhook
   └─ POST /webhooks/git

3. Catalog syncs and validates new resource
   └─ Parse YAML → Validate schema → Store in DynamoDB

4. Catalog broadcasts update via SSE
   └─ All connected Composers receive update

5. Composers invalidate cache and fetch new definitions
   └─ Ensures teams can use latest resources immediately
```

## 🏛️ Architecture Patterns

### **Service Communication**
- **Synchronous**: HTTP APIs for request/response patterns
- **Asynchronous**: SSE for real-time updates, message queues for deployments
- **Caching**: Redis-based caching at each layer for performance
- **Circuit Breakers**: Graceful degradation when services are unavailable

### **Data Consistency**
- **Eventually Consistent**: Catalog updates propagate asynchronously
- **Strong Consistency**: Deployment operations within orchestrator
- **Conflict Resolution**: Last-writer-wins for resource definitions
- **Version Control**: Semantic versioning prevents breaking changes

### **Security & Isolation**
- **Service-to-Service**: mTLS between all services
- **Team Isolation**: Namespace-based isolation in composers
- **RBAC**: Role-based access control at each service layer
- **Audit Logging**: Complete audit trail for all operations

## 📋 Component Deep Dive

### **CLI Tool**
```go
// CLI Architecture
cmd/
├── root.go           # Root command and global flags
├── generate.go       # Parse annotations and create resources
├── apply.go          # Deploy resources through composer
├── status.go         # Check deployment status
└── rollback.go       # Rollback deployments

internal/
├── annotations/      # Code annotation parsing
├── composer/         # Composer service client
└── templates/        # Resource template generation
```

### **Catalog Service**
```go
// Catalog Architecture
internal/
├── api/              # HTTP API layer
│   ├── handlers/     # Resource CRUD endpoints
│   └── middleware/   # Auth, CORS, rate limiting
├── storage/          # Storage abstraction layer
│   ├── dynamodb/     # DynamoDB implementation
│   └── cache/        # Redis caching layer
├── git/              # Git integration
│   ├── sync.go       # Repository synchronization
│   ├── webhook.go    # Git webhook handlers
│   └── parser.go     # YAML resource parsing
├── events/           # Real-time event system
│   ├── sse.go        # Server-Sent Events
│   └── broadcaster.go # Event broadcasting
└── validation/       # Resource validation
    ├── schema.go     # Schema validation
    └── policies.go   # Policy enforcement
```

**Key Design Decisions:**
- **Read-Heavy Optimization**: Aggressive caching, optimized for high read throughput
- **Git as Source of Truth**: All resources defined in Git, synced to database
- **Schema Validation**: Platform teams control resource schemas
- **Real-time Updates**: SSE ensures all services get immediate updates

### **Orchestrator Service**
```go
// Orchestrator Architecture
internal/
├── api/              # HTTP API layer
│   └── handlers/     # Deployment endpoints
├── deployment/       # Core deployment logic
│   ├── coordinator.go # Main coordination logic
│   ├── dependency.go # Dependency resolution
│   ├── queue.go      # Async deployment queue
│   └── status.go     # Status aggregation
├── engines/          # Engine abstraction
│   ├── interfaces.go # Engine interface definitions
│   ├── registry.go   # Engine registry
│   ├── crossplane/   # Crossplane engine
│   ├── pulumi/       # Pulumi engine
│   ├── terraform/    # Terraform engine
│   └── helm/         # Helm engine
├── catalog/          # Catalog service client
│   ├── client.go     # HTTP client for catalog
│   └── cache.go      # Local resource caching
└── gitops/           # GitOps integration
    ├── manager.go    # Git repository management
    ├── argocd.go     # ArgoCD integration
    └── manifest.go   # Manifest generation
```

**Key Design Decisions:**
- **Stateless Design**: All state managed by engines or external stores
- **Plugin Architecture**: Easy to add new deployment engines
- **Dependency Resolution**: Complex dependency graphs with cycle detection
- **Engine Abstraction**: Uniform interface across different IaC tools

### **Composer Service**
```go
// Composer Architecture
internal/
├── api/              # HTTP API layer
│   ├── handlers/     # Team-specific endpoints
│   └── team/         # Team isolation middleware
├── composition/      # Resource composition logic
│   ├── resolver.go   # Resolve compositions to resources
│   ├── validator.go  # Business logic validation
│   └── template.go   # Template rendering
├── catalog/          # Catalog service client
│   ├── client.go     # Catalog API client
│   └── cache.go      # Local catalog caching
├── orchestrator/     # Orchestrator service client
│   ├── client.go     # Orchestrator API client
│   └── deploy.go     # Deployment coordination
└── storage/          # Team composition storage
    ├── postgres/     # PostgreSQL for compositions
    └── cache/        # Redis for performance
```

**Key Design Decisions:**
- **Team Isolation**: Each team gets isolated namespace and resources
- **Business Logic Layer**: Team-specific validation and workflows
- **Composition Engine**: Combine catalog primitives into abstractions
- **API Gateway Pattern**: Team-specific API endpoints and authentication
```

## 🔧 Development Philosophy

### **Interface-Driven Design**
All services follow strict interface patterns for external dependencies:

```go
// Example: Catalog service interfaces
type ResourceStore interface {
    GetResource(ctx context.Context, name, version string) (*Resource, error)
    ListResources(ctx context.Context, filters *Filters) ([]*Resource, error)
    CreateResource(ctx context.Context, resource *Resource) error
}

type EventBroadcaster interface {
    Broadcast(ctx context.Context, event *Event) error
    Subscribe(ctx context.Context, filters *EventFilters) (<-chan *Event, error)
}

// Implementations are pluggable
func NewCatalogService(store ResourceStore, broadcaster EventBroadcaster) *CatalogService
```

### **Error Handling Strategy**
Centralized error handling with rich context:

```go
// Use shared error package across all services
return errors.New(errors.ErrorCodeResourceNotFound).
    Service("catalog").
    Operation("GetResource").
    Message("resource not found").
    Detail("resource_name", name).
    Detail("version", version).
    Build()
```

### **Configuration Management**
Environment-specific configuration with validation:

```yaml
# Each service follows consistent config patterns
service:
  name: "catalog"
  port: 8080

storage:
  type: "dynamodb"
  dynamodb:
    table_name: "nestor-catalog"
    region: "us-west-2"

cache:
  type: "redis"
  redis:
    url: "redis://localhost:6379"
    ttl: "5m"
```

## 🚀 Deployment Patterns

### **Service Deployment**
Each service deploys independently with Helm charts:

```bash
# Deploy catalog service
helm install nestor-catalog deployments/helm/catalog \
  --set config.storage.dynamodb.table_name=nestor-catalog-prod

# Deploy orchestrator service
helm install nestor-orchestrator deployments/helm/orchestrator \
  --set config.catalog.endpoint=https://catalog.nestor.svc.cluster.local

# Deploy composer per team
helm install team-alpha-composer deployments/helm/composer \
  --set team.name=alpha \
  --set team.namespace=team-alpha
```

### **Development Environment**
Local development with docker-compose:

```yaml
# docker-compose.yml
services:
  catalog:
    build: ./catalog
    ports: ["8080:8080"]
    depends_on: [dynamodb, redis]

  orchestrator:
    build: ./orchestrator
    ports: ["8081:8080"]
    depends_on: [catalog, redis]

  composer-alpha:
    build: ./composer
    ports: ["8082:8080"]
    environment:
      - TEAM_NAME=alpha
    depends_on: [catalog, orchestrator]
```

## 📊 Data Models

### **Catalog Resources**
```go
type ResourceDefinition struct {
    Metadata ResourceMetadata `json:"metadata"`
    Spec     ResourceSpec     `json:"spec"`
    Status   ResourceStatus   `json:"status"`
}

type ResourceMetadata struct {
    Name         string            `json:"name"`
    Version      string            `json:"version"`
    Provider     string            `json:"provider"`     // aws, gcp, azure
    Category     string            `json:"category"`     // database, compute, storage
    ResourceType string            `json:"resource_type"` // mysql, redis, s3
    Labels       map[string]string `json:"labels"`
}
```

### **Deployment Requests**
```go
type DeploymentRequest struct {
    ID          string               `json:"id"`
    Team        string               `json:"team"`
    Resources   []ResourceInstance   `json:"resources"`
    Dependencies []Dependency        `json:"dependencies"`
    Environment string              `json:"environment"`
}

type ResourceInstance struct {
    Name       string                 `json:"name"`
    CatalogRef string                 `json:"catalog_ref"` // "aws-rds-mysql:1.2.0"
    Config     map[string]interface{} `json:"config"`
    Engine     string                 `json:"engine"`      // "crossplane"
}
```

### **Team Compositions**
```go
type ComposedResource struct {
    Metadata CompositionMetadata `json:"metadata"`
    Spec     CompositionSpec     `json:"spec"`
}

type CompositionSpec struct {
    Description string             `json:"description"`
    Resources   []ComposedInstance `json:"resources"`
    Parameters  []Parameter        `json:"parameters"`
    Dependencies []Dependency      `json:"dependencies"`
}
```

## 🔍 Observability Strategy

### **Metrics Collection**
Each service exposes Prometheus metrics:

```go
// Service-specific metrics
var (
    resourcesServed = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "catalog_resources_served_total",
            Help: "Total number of resources served",
        },
        []string{"resource_type", "version"},
    )

    deploymentDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "orchestrator_deployment_duration_seconds",
            Help: "Deployment duration in seconds",
        },
        []string{"engine", "status"},
    )
)
```

### **Distributed Tracing**
OpenTelemetry integration across all services:

```go
// Trace requests across service boundaries
func (h *Handler) HandleDeployment(w http.ResponseWriter, r *http.Request) {
    ctx, span := tracer.Start(r.Context(), "handle_deployment")
    defer span.End()

    // Trace propagates to downstream services
    deployment, err := h.orchestrator.Deploy(ctx, request)
    if err != nil {
        span.SetStatus(codes.Error, err.Error())
        return
    }
}
```

### **Structured Logging**
Consistent logging across all services:

```go
logger.InfoContext(ctx, "deployment started",
    "deployment_id", req.ID,
    "team", req.Team,
    "resource_count", len(req.Resources),
    "environment", req.Environment,
)
```

## 🛡️ Security Architecture

### **Service-to-Service Communication**
- **mTLS**: All internal service communication uses mutual TLS
- **API Keys**: Service authentication via rotating API keys
- **Network Policies**: Kubernetes network policies for traffic control

### **Team Isolation**
- **Namespace Isolation**: Each team operates in isolated namespaces
- **RBAC**: Role-based access control at service and resource levels
- **Resource Quotas**: Prevent resource exhaustion by teams

### **Audit & Compliance**
- **Audit Logs**: All operations logged with full context
- **Change Tracking**: Git-based change tracking for all resources
- **Compliance Checks**: Automated policy validation

## 🎯 Scalability Considerations

### **Horizontal Scaling**
- **Stateless Services**: All services designed for horizontal scaling
- **Load Balancing**: Service mesh for intelligent traffic routing
- **Auto-scaling**: HPA based on metrics and queue depth

### **Performance Optimization**
- **Caching Strategy**: Multi-layer caching (Redis, in-memory, CDN)
- **Database Optimization**: Read replicas, connection pooling
- **Async Processing**: Queue-based processing for heavy operations

### **Multi-Region Support**
- **Catalog Replication**: Global catalog with regional caches
- **Engine Distribution**: Deploy engines close to target resources
- **Cross-Region Dependencies**: Handle cross-region resource dependencies

## 🔄 Migration & Evolution

### **Service Evolution**
- **API Versioning**: Semantic versioning for all service APIs
- **Backward Compatibility**: Maintain compatibility across versions
- **Feature Flags**: Safe feature rollout and rollback

### **Data Migration**
- **Schema Evolution**: Database schema migration strategies
- **Zero-Downtime**: Rolling updates with backward compatibility
- **Rollback Procedures**: Safe rollback for failed migrations

## 🎯 Success Metrics

### **Platform Health**
- **Service Availability**: 99.9% uptime for all core services
- **API Response Times**: <100ms for read operations, <500ms for writes
- **Deployment Success Rate**: >95% success rate for deployments

### **Team Productivity**
- **Self-Service Adoption**: % of infrastructure requests via self-service
- **Time to Provision**: Average time from request to ready resources
- **Developer Satisfaction**: Regular surveys and feedback collection

### **Platform Efficiency**
- **Resource Utilization**: Optimize cost through right-sizing
- **Operational Overhead**: Reduce manual intervention requirements
- **Compliance**: Automated policy compliance across all resources

---

**This architecture enables true platform-as-a-product delivery - providing teams with self-service capabilities while maintaining the governance and control that platform teams need.**
