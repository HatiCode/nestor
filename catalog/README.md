# Nestor Catalog Service

The Catalog Service is the **foundation layer** of the Nestor platform. It manages the global repository of infrastructure resource definitions, providing the low-level primitives that teams use to build their applications.

## 🏗️ Architecture Overview

The Catalog Service serves as the **single source of truth** for infrastructure resource definitions:

```
Platform Teams → Git Repositories → Catalog Service → (Orchestrator, Composers)
   (Define)         (Store)           (Serve)           (Consume)
```

**Key Responsibilities:**

- **Resource Definition Storage**: Centralized storage of infrastructure primitives
- **Git Synchronization**: Real-time sync from platform team repositories
- **Version Management**: Semantic versioning for all resource definitions
- **Real-time Updates**: Server-Sent Events for live catalog changes
- **Validation & Governance**: Platform team-controlled resource schemas
- **Discovery & Search**: API for finding and exploring available resources

## 📁 Directory Structure

```
catalog/
├── main.go                           # Application entry point
├── internal/                         # Private catalog implementation
│   ├── api/                          # HTTP API layer
│   │   ├── handlers/                 # HTTP request handlers
│   │   │   ├── resources.go          # Resource CRUD endpoints
│   │   │   ├── search.go             # Resource discovery endpoints
│   │   │   ├── versions.go           # Version management endpoints
│   │   │   ├── health.go             # Health check endpoints
│   │   │   └── sse.go                # Server-Sent Events
│   │   └── middleware/               # HTTP middleware
│   │
│   ├── storage/                      # Storage abstraction layer
│   │   ├── store.go                  # ComponentStore interface
│   │   ├── factory.go                # Storage factory
│   │   ├── dynamodb/                 # DynamoDB implementation
│   │   │   ├── client.go             # DynamoDB client wrapper
│   │   │   ├── component_store.go    # ComponentStore implementation
│   │   │   ├── config.go             # DynamoDB configuration
│   │   │   └── init.go               # Registration with factory
│   │   ├── memory/                   # In-memory implementation (testing)
│   │   └── cache/                    # Redis caching layer
│   │
│   ├── git/                          # Git integration
│   │   ├── sync.go                   # Repository synchronization
│   │   ├── webhook.go                # Git webhook handlers
│   │   ├── parser.go                 # YAML resource parsing
│   │   └── watcher.go                # File system watching
│   │
│   ├── events/                       # Real-time event system
│   │   ├── sse.go                    # Server-Sent Events server
│   │   ├── broadcaster.go            # Event broadcasting
│   │   └── types.go                  # Event type definitions
│   │
│   ├── validation/                   # Resource validation
│   │   ├── schema.go                 # Schema validation
│   │   ├── policies.go               # Policy enforcement
│   │   └── rules.go                  # Validation rule engine
│   │
│   └── observability/                # Metrics, logging, tracing
│       ├── metrics/                  # Prometheus metrics
│       ├── logging/                  # Structured logging
│       └── tracing/                  # Distributed tracing
│
├── pkg/                              # Public APIs
│   ├── api/                          # Client libraries
│   │   ├── client.go                 # HTTP API client
│   │   ├── sse_client.go             # SSE client for real-time updates
│   │   └── types.go                  # Request/response types
│   └── models/                       # Data models
│       ├── resource.go               # ResourceDefinition model
│       ├── version.go                # Version-related models
│       └── event.go                  # Event models for SSE
│
├── configs/                          # Configuration files
├── deployments/                      # K8s/Helm/Docker deployment manifests
├── examples/                         # Sample resource definitions
└── docs/                            # Architecture documentation
```

## 🚀 Core Concepts

### **Resource Definitions**

Infrastructure primitives managed by platform teams:

```yaml
# examples/aws-rds-mysql.yaml
apiVersion: catalog.nestor.io/v1
kind: ResourceDefinition
metadata:
  name: aws-rds-mysql
  version: "1.2.0"
  provider: aws
  category: database
  resource_type: mysql
  description: "AWS RDS MySQL database instance"
  maintainers:
    - platform-team@company.com
spec:
  supported_engines:
    - crossplane
    - pulumi
    - terraform

  required_inputs:
    - name: instance_class
      type: string
      description: "DB instance class (e.g., db.t3.micro)"
      validation:
        pattern: "^db\\.[a-z0-9]+\\.[a-z]+$"

    - name: allocated_storage
      type: integer
      description: "Allocated storage in GB"
      validation:
        min: 20
        max: 1000

  optional_inputs:
    - name: backup_retention_period
      type: integer
      default: 7
      description: "Backup retention period in days"

  outputs:
    - name: endpoint
      type: string
      description: "Database connection endpoint"

    - name: port
      type: integer
      description: "Database port"

  engine_configs:
    crossplane:
      composition_ref: "aws-rds-mysql-composition"
      version: "v1.2.0"

    pulumi:
      program_template: "aws-rds-mysql-pulumi"
      language: "typescript"

    terraform:
      module_source: "terraform-aws-modules/rds/aws"
      module_version: "~> 5.0"
```

### **Version Management**

Semantic versioning with backward compatibility:

```
aws-rds-mysql:1.0.0  # Initial release
aws-rds-mysql:1.1.0  # Added backup_retention_period (minor)
aws-rds-mysql:1.2.0  # Added multi-az support (minor)
aws-rds-mysql:2.0.0  # Changed instance_class validation (major)
```

### **Real-time Updates**

Server-Sent Events for live catalog synchronization:

```javascript
// Example SSE client usage
const eventSource = new EventSource("/api/v1/events");

eventSource.addEventListener("resource.created", (event) => {
  const resource = JSON.parse(event.data);
  console.log("New resource available:", resource.metadata.name);
});

eventSource.addEventListener("resource.updated", (event) => {
  const resource = JSON.parse(event.data);
  console.log("Resource updated:", resource.metadata.name);
});
```

## 🎯 API Endpoints

### **Resource Management**

```http
# List all resources
GET /api/v1/resources
GET /api/v1/resources?provider=aws
GET /api/v1/resources?category=database

# Get specific resource
GET /api/v1/resources/{name}                # Latest version
GET /api/v1/resources/{name}/{version}      # Specific version

# Version management
GET /api/v1/resources/{name}/versions       # List all versions
GET /api/v1/resources/{name}/latest         # Get latest version info

# Search and discovery
GET /api/v1/search?q=mysql                  # Search resources
GET /api/v1/search?q=database&provider=aws  # Filtered search
```

### **Real-time Updates**

```http
# Server-Sent Events stream
GET /api/v1/events
GET /api/v1/events?filter=provider:aws      # Filtered events
GET /api/v1/events?filter=category:database # Category-specific events
```

### **Health & Status**

```http
GET /health                                 # Service health
GET /ready                                  # Readiness probe
GET /metrics                                # Prometheus metrics
GET /api/v1/status                          # Catalog status
```

## 🔧 Configuration

```yaml
# configs/production.yaml
service:
  name: catalog
  port: 8080
  environment: production

logging:
  level: info
  format: json

storage:
  type: dynamodb
  dynamodb:
    table_name: nestor-catalog-prod
    region: us-west-2
    read_capacity: 10
    write_capacity: 5
    enable_point_in_time_recovery: true

cache:
  type: redis
  redis:
    url: redis://redis-cluster.nestor.svc.cluster.local:6379
    pool_size: 10
    ttl: 5m

git:
  repositories:
    - name: platform-resources
      url: https://github.com/company/platform-resources.git
      path: resources/
      branch: main
      poll_interval: 60s
      webhook_secret: "${GIT_WEBHOOK_SECRET}"

    - name: aws-resources
      url: https://github.com/company/aws-resources.git
      path: definitions/
      branch: main
      poll_interval: 60s

sse:
  enabled: true
  max_connections: 1000
  heartbeat_interval: 30s
  buffer_size: 100

validation:
  strict_mode: true
  policy_enforcement: true
  schema_validation: true
```

## 🚀 Getting Started

### **Local Development**

```bash
# Start dependencies
make docker-up

# Start catalog service
make dev-catalog

# Service will be available at http://localhost:8080
```

### **Docker Deployment**

```bash
# Build and run
make docker-build
docker run -p 8080:8080 \
  -e STORAGE_TYPE=dynamodb \
  -e DYNAMODB_ENDPOINT=http://dynamodb:8000 \
  nestor/catalog:dev
```

### **Kubernetes Deployment**

```bash
# Using Helm
helm install nestor-catalog deployments/helm \
  --set config.storage.dynamodb.table_name=nestor-catalog-prod \
  --set config.git.repositories[0].url=https://github.com/company/platform-resources.git
```

## 📊 Monitoring & Observability

### **Key Metrics**

- `catalog_resources_total` - Total resources in catalog
- `catalog_git_sync_duration` - Git synchronization time
- `catalog_api_requests_total` - API request count and latency
- `catalog_sse_connections` - Active SSE connections
- `catalog_validation_errors` - Validation error count

### **Health Checks**

- **Liveness**: Service process health
- **Readiness**: Storage and cache connectivity
- **Dependencies**: Git repository accessibility

### **Logging**

Structured logging with correlation IDs:

```json
{
  "timestamp": "2024-01-15T10:30:45Z",
  "level": "info",
  "service": "catalog",
  "operation": "sync_repository",
  "repository": "platform-resources",
  "resources_updated": 5,
  "duration_ms": 1250,
  "trace_id": "abc123"
}
```

## 🔒 Security

### **Authentication**

- **Service-to-Service**: mTLS for internal communication
- **External Access**: JWT tokens for API access
- **Git Integration**: SSH keys or personal access tokens

### **Authorization**

- **Read Access**: All authenticated services can read catalog
- **Write Access**: Only platform teams can modify resources
- **Git Integration**: Platform team repository access controls

### **Audit Logging**

Complete audit trail for all resource changes:

```json
{
  "event": "resource.updated",
  "resource": "aws-rds-mysql",
  "version": "1.2.0",
  "user": "platform-team@company.com",
  "timestamp": "2024-01-15T10:30:45Z",
  "changes": ["spec.optional_inputs"],
  "git_commit": "abc123def456"
}
```

## 🎯 Integration Patterns

### **With Orchestrator Service**

```go
// Orchestrator queries catalog for resource definitions
catalogClient := catalog.NewClient(config.CatalogEndpoint)

resource, err := catalogClient.GetResource(ctx, "aws-rds-mysql", "1.2.0")
if err != nil {
    return fmt.Errorf("failed to get resource definition: %w", err)
}

// Use resource definition for deployment
deployment := orchestrator.CreateDeployment(resource, userConfig)
```

### **With Composer Service**

```go
// Composer subscribes to catalog updates
eventStream, err := catalogClient.Subscribe(ctx, &catalog.EventFilters{
    Provider: "aws",
    Category: "database",
})

for event := range eventStream {
    switch event.Type {
    case "resource.created":
        composer.InvalidateCache(event.ResourceName)
    case "resource.updated":
        composer.RefreshCompositions(event.ResourceName)
    }
}
```

## 🚨 Error Handling

### **Resource Not Found**

```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Resource not found",
    "service": "catalog",
    "details": {
      "resource_name": "aws-rds-mysql",
      "version": "2.0.0"
    },
    "trace_id": "abc123"
  }
}
```

### **Validation Errors**

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "Resource validation failed",
    "service": "catalog",
    "details": {
      "resource_name": "aws-rds-mysql",
      "validation_errors": [
        {
          "field": "spec.required_inputs[0].name",
          "message": "field name must be snake_case"
        }
      ]
    }
  }
}
```

## 🎯 Key Design Decisions

### **Global Catalog**

- **Single source of truth** for all infrastructure primitives
- **No team isolation** at the catalog level - resources are globally available
- **Platform team ownership** ensures consistency and governance

### **Git as Source of Truth**

- **All resources defined in Git** repositories managed by platform teams
- **Real-time synchronization** from Git to database storage
- **Webhook-driven updates** for immediate catalog updates

### **Read-Heavy Optimization**

- **Aggressive caching** with Redis for frequently accessed resources
- **DynamoDB optimized** for high read throughput
- **SSE for real-time updates** without polling overhead

### **Version Immutability**

- **Resource versions are immutable** once published
- **Semantic versioning enforced** for predictable compatibility
- **Deprecation workflow** for managing resource lifecycle

---

**The Catalog Service provides the foundation that enables the entire Nestor platform - serving as the authoritative source for infrastructure primitives that teams compose into their applications.**
