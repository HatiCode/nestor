# Nestor Platform - Service Interfaces Documentation

This document defines the key interfaces and contracts between the three core services: Catalog, Orchestrator, and Composers. Keep this in sync with actual implementations.

## 🏗️ Service Architecture Overview

```
CLI → Composer → Orchestrator → Catalog
     (Team APIs) (Deployment)  (Resources)
```

Each service exposes well-defined APIs and follows consistent patterns for authentication, error handling, and data exchange.

## 📚 Catalog Service Interfaces

The Catalog service provides the foundation layer - managing infrastructure resource definitions and real-time updates.

### **Core Storage Interface**
```go
// catalog/internal/storage/interfaces.go
type ResourceStore interface {
    // Resource CRUD operations
    GetResource(ctx context.Context, name, version string) (*models.ResourceDefinition, error)
    GetLatestResource(ctx context.Context, name string) (*models.ResourceDefinition, error)
    ListResources(ctx context.Context, req *ListResourcesRequest) (*ListResourcesResponse, error)
    CreateResource(ctx context.Context, resource *models.ResourceDefinition) error
    UpdateResource(ctx context.Context, resource *models.ResourceDefinition) error
    DeleteResource(ctx context.Context, name, version string) error

    // Version management
    GetResourceVersions(ctx context.Context, name string) ([]*models.ResourceVersion, error)
    ResourceExists(ctx context.Context, name, version string) (bool, error)

    // Search and discovery
    SearchResources(ctx context.Context, req *SearchResourcesRequest) (*SearchResourcesResponse, error)
    FindResourcesByProvider(ctx context.Context, provider string) ([]*models.ResourceDefinition, error)
    FindResourcesByCategory(ctx context.Context, category string) ([]*models.ResourceDefinition, error)
}

type GitSynchronizer interface {
    // Git repository synchronization
    SyncRepository(ctx context.Context, repo *GitRepository) error
    ParseResourceDefinitions(ctx context.Context, repoPath string) ([]*models.ResourceDefinition, error)
    ValidateResourceDefinition(ctx context.Context, resource *models.ResourceDefinition) error

    // Webhook handling
    HandleWebhook(ctx context.Context, event *GitWebhookEvent) error
}

type EventBroadcaster interface {
    // Real-time event broadcasting
    Broadcast(ctx context.Context, event *Event) error
    Subscribe(ctx context.Context, filters *EventFilters) (<-chan *Event, error)
    Unsubscribe(ctx context.Context, subscription string) error
}
```

### **REST API Endpoints**
```http
# Resource Management
GET    /api/v1/resources                    # List all resources
GET    /api/v1/resources/{name}             # Get specific resource (latest version)
GET    /api/v1/resources/{name}/{version}   # Get specific version
GET    /api/v1/resources/{name}/versions    # List all versions of resource

# Search and Discovery
GET    /api/v1/search?q={query}             # Search resources
GET    /api/v1/resources?provider={aws}     # Filter by provider
GET    /api/v1/resources?category={database} # Filter by category

# Real-time Updates
GET    /api/v1/events                       # Server-Sent Events stream
```

### **Data Models**
```go
// catalog/pkg/models/resource.go
type ResourceDefinition struct {
    Metadata ResourceMetadata `json:"metadata"`
    Spec     ResourceSpec     `json:"spec"`
    Status   ResourceStatus   `json:"status"`
}

type ResourceMetadata struct {
    Name         string            `json:"name" validate:"required,dns1123"`
    Version      string            `json:"version" validate:"required,semver"`
    Provider     string            `json:"provider" validate:"required"`      // aws, gcp, azure, k8s
    Category     string            `json:"category" validate:"required"`      // database, compute, storage
    ResourceType string            `json:"resource_type" validate:"required"` // mysql, redis, s3

    Description  string            `json:"description"`
    Maintainers  []string          `json:"maintainers"`
    Documentation []DocLink        `json:"documentation"`

    // Git source tracking
    GitRepository string           `json:"git_repository"`
    GitPath       string           `json:"git_path"`
    GitCommit     string           `json:"git_commit"`

    // Metadata
    Labels        map[string]string `json:"labels"`
    Annotations   map[string]string `json:"annotations"`
    CreatedAt     time.Time         `json:"created_at"`
    UpdatedAt     time.Time         `json:"updated_at"`
}

type ResourceSpec struct {
    // Engine support
    SupportedEngines []string              `json:"supported_engines"` // crossplane, pulumi, terraform, helm

    // Input/Output specifications
    RequiredInputs   []InputSpec           `json:"required_inputs"`
    OptionalInputs   []InputSpec           `json:"optional_inputs"`
    Outputs          []OutputSpec          `json:"outputs"`

    // Engine-specific configurations
    EngineConfigs    map[string]EngineConfig `json:"engine_configs"`

    // Dependencies and conflicts
    Dependencies     []string              `json:"dependencies"`     // Other resources this depends on
    ConflictsWith    []string              `json:"conflicts_with"`   // Resources that conflict with this one
}
```

## 🎼 Orchestrator Service Interfaces

The Orchestrator coordinates complex deployments across multiple engines with dependency resolution.

### **Core Deployment Interface**
```go
// orchestrator/internal/deployment/interfaces.go
type DeploymentCoordinator interface {
    // Deployment lifecycle
    CreateDeployment(ctx context.Context, req *DeploymentRequest) (*Deployment, error)
    GetDeployment(ctx context.Context, id string) (*Deployment, error)
    ListDeployments(ctx context.Context, filters *DeploymentFilters) ([]*Deployment, error)
    CancelDeployment(ctx context.Context, id string) error
    RollbackDeployment(ctx context.Context, id string, targetVersion string) error

    // Status and monitoring
    GetDeploymentStatus(ctx context.Context, id string) (*DeploymentStatus, error)
    WatchDeploymentStatus(ctx context.Context, id string) (<-chan *DeploymentStatus, error)
}

type DependencyResolver interface {
    // Dependency management
    ResolveDependencies(ctx context.Context, resources []ResourceInstance) (*DependencyGraph, error)
    ValidateDependencies(ctx context.Context, graph *DependencyGraph) error
    GetDeploymentOrder(ctx context.Context, graph *DependencyGraph) ([][]ResourceInstance, error)
    DetectCircularDependencies(ctx context.Context, graph *DependencyGraph) error
}

type EngineRegistry interface {
    // Engine management
    RegisterEngine(ctx context.Context, engine DeploymentEngine) error
    GetEngine(ctx context.Context, name string) (DeploymentEngine, error)
    ListEngines(ctx context.Context) ([]DeploymentEngine, error)
    IsEngineHealthy(ctx context.Context, name string) (bool, error)
    GetEngineCapabilities(ctx context.Context, name string) (*EngineCapabilities, error)
}

type DeploymentEngine interface {
    // Engine operations
    Deploy(ctx context.Context, resource *ResourceInstance) (*DeploymentResult, error)
    GetStatus(ctx context.Context, deploymentID string) (*DeploymentStatus, error)
    Update(ctx context.Context, deploymentID string, resource *ResourceInstance) error
    Delete(ctx context.Context, deploymentID string) error

    // Engine metadata
    Name() string
    Version() string
    SupportedResourceTypes() []string
    Capabilities() *EngineCapabilities
    HealthCheck(ctx context.Context) error
}
```

### **REST API Endpoints**
```http
# Deployment Management
POST   /api/v1/deployments                  # Create new deployment
GET    /api/v1/deployments                  # List deployments
GET    /api/v1/deployments/{id}             # Get deployment details
DELETE /api/v1/deployments/{id}             # Cancel deployment
POST   /api/v1/deployments/{id}/rollback    # Rollback deployment

# Status and Monitoring
GET    /api/v1/deployments/{id}/status      # Get deployment status
GET    /api/v1/deployments/{id}/logs        # Get deployment logs
GET    /api/v1/deployments/{id}/events      # Get deployment events

# Engine Management
GET    /api/v1/engines                      # List available engines
GET    /api/v1/engines/{name}               # Get engine details
GET    /api/v1/engines/{name}/health        # Check engine health
```

### **Data Models**
```go
// orchestrator/pkg/models/deployment.go
type DeploymentRequest struct {
    ID          string              `json:"id" validate:"required"`
    Team        string              `json:"team" validate:"required"`
    Environment string              `json:"environment" validate:"required"`

    // Resources to deploy
    Resources   []ResourceInstance  `json:"resources" validate:"required,min=1"`

    // Dependency specification
    Dependencies []Dependency       `json:"dependencies"`

    // Deployment configuration
    Config      DeploymentConfig    `json:"config"`

    // Metadata
    Labels      map[string]string   `json:"labels"`
    Annotations map[string]string   `json:"annotations"`
}

type ResourceInstance struct {
    Name         string                 `json:"name" validate:"required"`
    CatalogRef   string                 `json:"catalog_ref" validate:"required"` // "aws-rds-mysql:1.2.0"
    Engine       string                 `json:"engine" validate:"required"`      // "crossplane"
    Config       map[string]interface{} `json:"config"`

    // Dependencies within this deployment
    DependsOn    []string               `json:"depends_on"`

    // Output mapping
    Outputs      map[string]string      `json:"outputs"`
}

type Deployment struct {
    ID          string              `json:"id"`
    Status      DeploymentStatus    `json:"status"`
    Resources   []DeployedResource  `json:"resources"`
    Dependencies *DependencyGraph   `json:"dependencies"`

    CreatedAt   time.Time           `json:"created_at"`
    UpdatedAt   time.Time           `json:"updated_at"`
    CompletedAt *time.Time          `json:"completed_at,omitempty"`
}

type DeploymentStatus string

const (
    DeploymentStatusPending    DeploymentStatus = "pending"
    DeploymentStatusRunning    DeploymentStatus = "running"
    DeploymentStatusSucceeded  DeploymentStatus = "succeeded"
    DeploymentStatusFailed     DeploymentStatus = "failed"
    DeploymentStatusCancelling DeploymentStatus = "cancelling"
    DeploymentStatusCancelled  DeploymentStatus = "cancelled"
)
```

## 🎵 Composer Service Interfaces

The Composer enables teams to create business-focused abstractions from catalog primitives and coordinates with the orchestrator for deployments.

### **Core Composition Interface**
```go
// composer/internal/composition/interfaces.go
type CompositionManager interface {
    // Composition lifecycle
    CreateComposition(ctx context.Context, comp *ComposedResource) error
    GetComposition(ctx context.Context, team, name string) (*ComposedResource, error)
    ListCompositions(ctx context.Context, team string) ([]*ComposedResource, error)
    UpdateComposition(ctx context.Context, comp *ComposedResource) error
    DeleteComposition(ctx context.Context, team, name string) error

    // Composition validation
    ValidateComposition(ctx context.Context, comp *ComposedResource) (*ValidationResult, error)
    ResolveComposition(ctx context.Context, comp *ComposedResource, params map[string]interface{}) (*ResolvedComposition, error)
}

type DeploymentManager interface {
    // Team deployment operations
    Deploy(ctx context.Context, req *TeamDeploymentRequest) (*TeamDeployment, error)
    GetDeployment(ctx context.Context, team, id string) (*TeamDeployment, error)
    ListDeployments(ctx context.Context, team string, filters *DeploymentFilters) ([]*TeamDeployment, error)
    CancelDeployment(ctx context.Context, team, id string) error

    // Status monitoring
    GetDeploymentStatus(ctx context.Context, team, id string) (*TeamDeploymentStatus, error)
    WatchDeploymentStatus(ctx context.Context, team, id string) (<-chan *TeamDeploymentStatus, error)
}

type CatalogClient interface {
    // Catalog service integration
    GetResource(ctx context.Context, name, version string) (*models.ResourceDefinition, error)
    ListResources(ctx context.Context, filters *ResourceFilters) ([]*models.ResourceDefinition, error)
    SearchResources(ctx context.Context, query string) ([]*models.ResourceDefinition, error)

    // Cache management
    InvalidateCache(ctx context.Context, resourceName string) error
    RefreshCache(ctx context.Context) error
}

type OrchestratorClient interface {
    // Orchestrator service integration
    CreateDeployment(ctx context.Context, req *DeploymentRequest) (*Deployment, error)
    GetDeployment(ctx context.Context, id string) (*Deployment, error)
    CancelDeployment(ctx context.Context, id string) error

    // Status monitoring
    GetDeploymentStatus(ctx context.Context, id string) (*DeploymentStatus, error)
    WatchDeploymentStatus(ctx context.Context, id string) (<-chan *DeploymentStatus, error)
}
```

### **REST API Endpoints**
```http
# Team-specific composition management
GET    /api/v1/teams/{team}/compositions          # List team compositions
POST   /api/v1/teams/{team}/compositions          # Create composition
GET    /api/v1/teams/{team}/compositions/{name}   # Get composition
PUT    /api/v1/teams/{team}/compositions/{name}   # Update composition
DELETE /api/v1/teams/{team}/compositions/{name}   # Delete composition

# Team deployment operations
POST   /api/v1/teams/{team}/deploy                # Deploy composition
GET    /api/v1/teams/{team}/deployments           # List team deployments
GET    /api/v1/teams/{team}/deployments/{id}      # Get deployment
DELETE /api/v1/teams/{team}/deployments/{id}      # Cancel deployment

# Resource discovery (filtered by team permissions)
GET    /api/v1/teams/{team}/catalog/resources     # List available resources
GET    /api/v1/teams/{team}/catalog/search        # Search resources
```

### **Data Models**
```go
// composer/pkg/models/composition.go
type ComposedResource struct {
    Metadata CompositionMetadata `json:"metadata"`
    Spec     CompositionSpec     `json:"spec"`
    Status   CompositionStatus   `json:"status"`
}

type CompositionMetadata struct {
    Name        string            `json:"name" validate:"required,dns1123"`
    Team        string            `json:"team" validate:"required"`
    Version     string            `json:"version" validate:"required,semver"`
    Description string            `json:"description"`

    // Composition metadata
    Labels      map[string]string `json:"labels"`
    Annotations map[string]string `json:"annotations"`

    // Timestamps
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

type CompositionSpec struct {
    // Resource composition
    Resources    []ComposedInstance `json:"resources" validate:"required,min=1"`

    // Template parameters
    Parameters   []Parameter        `json:"parameters"`

    // Internal dependencies
    Dependencies []Dependency       `json:"dependencies"`

    // Deployment configuration
    DeploymentConfig DeploymentConfig `json:"deployment_config"`

    // Environments
    Environments map[string]EnvironmentConfig `json:"environments"`
}

type ComposedInstance struct {
    Name        string                 `json:"name" validate:"required"`
    CatalogRef  string                 `json:"catalog_ref" validate:"required"` // "aws-rds-mysql:1.2.0"

    // Configuration with templating support
    Config      map[string]interface{} `json:"config"`

    // Conditional inclusion
    Condition   string                 `json:"condition,omitempty"`

    // Engine preference
    PreferredEngine string             `json:"preferred_engine,omitempty"`
}

type TeamDeploymentRequest struct {
    CompositionName string                 `json:"composition_name" validate:"required"`
    Parameters      map[string]interface{} `json:"parameters"`
    Environment     string                 `json:"environment" validate:"required"`

    // Deployment metadata
    Labels          map[string]string      `json:"labels"`
    Annotations     map[string]string      `json:"annotations"`
}
```

## 🔄 Inter-Service Communication Patterns

### **Service Discovery**
```go
// All services use consistent service discovery
type ServiceConfig struct {
    Name     string `json:"name"`
    Endpoint string `json:"endpoint"`
    Timeout  time.Duration `json:"timeout"`
    Retries  int    `json:"retries"`
}

// Example configuration
catalog:
  endpoint: "https://catalog.nestor.svc.cluster.local"
  timeout: "30s"
  retries: 3

orchestrator:
  endpoint: "https://orchestrator.nestor.svc.cluster.local"
  timeout: "60s"
  retries: 5
```

### **Authentication & Authorization**
```go
// Consistent auth patterns across services
type AuthConfig struct {
    Method string `json:"method"` // "jwt", "mtls", "api_key"

    // JWT configuration
    JWT struct {
        Secret   string `json:"secret"`
        Issuer   string `json:"issuer"`
        Audience string `json:"audience"`
    } `json:"jwt,omitempty"`

    // mTLS configuration
    MTLS struct {
        CertFile   string `json:"cert_file"`
        KeyFile    string `json:"key_file"`
        CAFile     string `json:"ca_file"`
    } `json:"mtls,omitempty"`
}
```

### **Error Handling**
```go
// Standardized error responses across all services
type APIError struct {
    Code    string `json:"code"`           // "RESOURCE_NOT_FOUND"
    Message string `json:"message"`        // Human-readable message
    Service string `json:"service"`        // "catalog"
    Details map[string]interface{} `json:"details,omitempty"`
    TraceID string `json:"trace_id"`       // For debugging
}

// Common error codes
const (
    ErrorCodeResourceNotFound    = "RESOURCE_NOT_FOUND"
    ErrorCodeValidationFailed    = "VALIDATION_FAILED"
    ErrorCodeDeploymentFailed    = "DEPLOYMENT_FAILED"
    ErrorCodeUnauthorized        = "UNAUTHORIZED"
    ErrorCodeForbidden          = "FORBIDDEN"
    ErrorCodeInternalError      = "INTERNAL_ERROR"
    ErrorCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)
```

### **Event Patterns**
```go
// Consistent event structure across services
type Event struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`          // "resource.created", "deployment.completed"
    Source    string                 `json:"source"`        // "catalog", "orchestrator", "composer"
    Subject   string                 `json:"subject"`       // Resource/deployment identifier
    Data      map[string]interface{} `json:"data"`
    Timestamp time.Time              `json:"timestamp"`
    TraceID   string                 `json:"trace_id"`
}

// Event types for each service
const (
    // Catalog events
    EventResourceCreated   = "resource.created"
    EventResourceUpdated   = "resource.updated"
    EventResourceDeleted   = "resource.deleted"

    // Orchestrator events
    EventDeploymentStarted   = "deployment.started"
    EventDeploymentCompleted = "deployment.completed"
    EventDeploymentFailed    = "deployment.failed"

    // Composer events
    EventCompositionCreated = "composition.created"
    EventCompositionUpdated = "composition.updated"
    EventTeamDeploymentStarted = "team_deployment.started"
)
```

## 🖥️ CLI Integration Interfaces

The CLI integrates primarily with the Composer service for team operations.

### **CLI Command Structure**
```go
// cli/internal/commands/interfaces.go
type ComposerClient interface {
    // Composition operations
    ListCompositions(ctx context.Context, team string) ([]*ComposedResource, error)
    GetComposition(ctx context.Context, team, name string) (*ComposedResource, error)
    CreateComposition(ctx context.Context, comp *ComposedResource) error

    // Deployment operations
    Deploy(ctx context.Context, req *TeamDeploymentRequest) (*TeamDeployment, error)
    GetDeploymentStatus(ctx context.Context, team, id string) (*TeamDeploymentStatus, error)
    ListDeployments(ctx context.Context, team string) ([]*TeamDeployment, error)

    // Resource discovery
    ListAvailableResources(ctx context.Context, team string) ([]*models.ResourceDefinition, error)
    SearchResources(ctx context.Context, team, query string) ([]*models.ResourceDefinition, error)
}

type AnnotationParser interface {
    // Code annotation parsing
    ParseFile(ctx context.Context, filePath string) ([]*Annotation, error)
    ParseDirectory(ctx context.Context, dirPath string) ([]*Annotation, error)
    ValidateAnnotations(ctx context.Context, annotations []*Annotation) error

    // Template generation
    GenerateComposition(ctx context.Context, annotations []*Annotation) (*ComposedResource, error)
    GenerateDeploymentRequest(ctx context.Context, annotations []*Annotation, params map[string]interface{}) (*TeamDeploymentRequest, error)
}
```

### **CLI Commands**
```bash
# Composition management
nestor compose list                              # List team compositions
nestor compose create --file web-app.yaml       # Create composition
nestor compose get web-app                      # Get composition details
nestor compose update web-app --file updated.yaml # Update composition

# Code annotation workflow
nestor generate                                 # Parse annotations and generate compositions
nestor apply                                   # Deploy based on annotations
nestor apply --dry-run                        # Show what would be deployed
nestor apply --environment staging            # Deploy to specific environment

# Deployment management
nestor deploy web-app --size large             # Deploy with parameters
nestor status                                  # Show all deployments
nestor status web-app-prod                    # Show specific deployment
nestor logs web-app-prod                      # Show deployment logs
nestor rollback web-app-prod                  # Rollback deployment

# Resource discovery
nestor catalog list                           # List available resources
nestor catalog search database                # Search for database resources
nestor catalog get aws-rds-mysql             # Get resource details
```

## 🔧 Configuration Patterns

### **Service Configuration Structure**
```go
// Each service follows consistent config patterns
type ServiceConfig struct {
    // Service identity
    Service ServiceInfo `json:"service"`

    // HTTP server configuration
    Server ServerConfig `json:"server"`

    // Storage configuration
    Storage StorageConfig `json:"storage"`

    // Cache configuration
    Cache CacheConfig `json:"cache"`

    // Logging configuration
    Logging LoggingConfig `json:"logging"`

    // Service-specific configuration
    // Catalog: Git, SSE
    // Orchestrator: Engines, GitOps
    // Composer: Team, Authentication
}

type ServiceInfo struct {
    Name        string `json:"name"`
    Version     string `json:"version"`
    Environment string `json:"environment"`
    Region      string `json:"region"`
}

type ServerConfig struct {
    Host         string        `json:"host"`
    Port         int           `json:"port"`
    ReadTimeout  time.Duration `json:"read_timeout"`
    WriteTimeout time.Duration `json:"write_timeout"`
    IdleTimeout  time.Duration `json:"idle_timeout"`
}
```

## 🧪 Testing Patterns

### **Interface Testing**
```go
// All interfaces should have comprehensive test suites
func TestResourceStore(t *testing.T) {
    // Test with multiple implementations
    implementations := []struct {
        name  string
        store ResourceStore
    }{
        {"memory", memory.NewResourceStore()},
        {"dynamodb", dynamodb.NewResourceStore(config)},
    }

    for _, impl := range implementations {
        t.Run(impl.name, func(t *testing.T) {
            testResourceStoreCRUD(t, impl.store)
            testResourceStoreSearch(t, impl.store)
            testResourceStoreVersioning(t, impl.store)
        })
    }
}

// Contract testing between services
func TestCatalogOrchestratorContract(t *testing.T) {
    // Mock catalog service
    catalogServer := httptest.NewServer(mockCatalogHandler())
    defer catalogServer.Close()

    // Real orchestrator with mocked catalog
    orchestrator := NewOrchestrator(OrchestratorConfig{
        CatalogEndpoint: catalogServer.URL,
    })

    // Test the interaction
    deployment, err := orchestrator.CreateDeployment(ctx, deploymentRequest)
    assert.NoError(t, err)
    assert.NotNil(t, deployment)
}
```

### **Mock Implementations**
```go
// Provide mock implementations for testing
type MockResourceStore struct {
    resources map[string]*models.ResourceDefinition
    mutex     sync.RWMutex
}

func (m *MockResourceStore) GetResource(ctx context.Context, name, version string) (*models.ResourceDefinition, error) {
    m.mutex.RLock()
    defer m.mutex.RUnlock()

    key := fmt.Sprintf("%s:%s", name, version)
    resource, exists := m.resources[key]
    if !exists {
        return nil, errors.New(errors.ErrorCodeResourceNotFound).
            Message("resource not found").
            Detail("name", name).
            Detail("version", version).
            Build()
    }

    return resource, nil
}
```

## 📊 Metrics & Observability Interfaces

### **Metrics Collection**
```go
// Standard metrics interface across all services
type MetricsCollector interface {
    // Counter metrics
    IncCounter(name string, labels map[string]string)

    // Histogram metrics
    RecordDuration(name string, duration time.Duration, labels map[string]string)

    // Gauge metrics
    SetGauge(name string, value float64, labels map[string]string)

    // Service health
    RecordHealthCheck(service string, healthy bool)
}

// Service-specific metrics
// Catalog Service
catalogMetrics := []Metric{
    {Name: "catalog_resources_total", Type: "counter", Help: "Total resources in catalog"},
    {Name: "catalog_requests_total", Type: "counter", Help: "Total API requests"},
    {Name: "catalog_git_sync_duration", Type: "histogram", Help: "Git sync duration"},
    {Name: "catalog_sse_connections", Type: "gauge", Help: "Active SSE connections"},
}

// Orchestrator Service
orchestratorMetrics := []Metric{
    {Name: "orchestrator_deployments_total", Type: "counter", Help: "Total deployments"},
    {Name: "orchestrator_deployment_duration", Type: "histogram", Help: "Deployment duration"},
    {Name: "orchestrator_engine_requests", Type: "counter", Help: "Engine requests by type"},
    {Name: "orchestrator_dependency_resolution_time", Type: "histogram", Help: "Dependency resolution time"},
}

// Composer Service
composerMetrics := []Metric{
    {Name: "composer_compositions_total", Type: "counter", Help: "Total compositions by team"},
    {Name: "composer_team_deployments", Type: "counter", Help: "Team deployments"},
    {Name: "composer_composition_resolution_time", Type: "histogram", Help: "Composition resolution time"},
    {Name: "composer_active_teams", Type: "gauge", Help: "Number of active teams"},
}
```

### **Health Check Interface**
```go
// Consistent health checking across services
type HealthChecker interface {
    // Service health
    CheckHealth(ctx context.Context) *HealthStatus

    // Dependency health
    CheckDependencies(ctx context.Context) map[string]*HealthStatus

    // Readiness check
    CheckReadiness(ctx context.Context) *ReadinessStatus
}

type HealthStatus struct {
    Status    string                 `json:"status"`    // "healthy", "unhealthy", "degraded"
    Timestamp time.Time              `json:"timestamp"`
    Details   map[string]interface{} `json:"details,omitempty"`
    Error     string                 `json:"error,omitempty"`
}

// Health check endpoints
GET /health      # Overall service health
GET /ready       # Readiness probe (Kubernetes)
GET /live        # Liveness probe (Kubernetes)
```

## 🔒 Security Interface Patterns

### **Authentication Interface**
```go
type Authenticator interface {
    // Token validation
    ValidateToken(ctx context.Context, token string) (*AuthContext, error)

    // Service authentication
    AuthenticateService(ctx context.Context, credentials *ServiceCredentials) (*ServiceAuth, error)

    // Token refresh
    RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
}

type AuthContext struct {
    UserID      string            `json:"user_id"`
    Team        string            `json:"team"`
    Roles       []string          `json:"roles"`
    Permissions []string          `json:"permissions"`
    Claims      map[string]interface{} `json:"claims"`
}
```

### **Authorization Interface**
```go
type Authorizer interface {
    // Resource-level authorization
    CanAccessResource(ctx context.Context, auth *AuthContext, resource, action string) (bool, error)

    // Team-level authorization
    CanAccessTeam(ctx context.Context, auth *AuthContext, team, action string) (bool, error)

    // Service-level authorization
    CanAccessService(ctx context.Context, auth *AuthContext, service, action string) (bool, error)
}

// Authorization actions
const (
    ActionRead   = "read"
    ActionWrite  = "write"
    ActionDelete = "delete"
    ActionDeploy = "deploy"
    ActionAdmin  = "admin"
)
```

## 🔧 Client Library Interfaces

### **Go Client Libraries**
```go
// Each service provides a Go client library
// catalog/pkg/client/client.go
type CatalogClient struct {
    endpoint string
    auth     Authenticator
    timeout  time.Duration
}

func NewCatalogClient(config *ClientConfig) *CatalogClient

func (c *CatalogClient) GetResource(ctx context.Context, name, version string) (*models.ResourceDefinition, error)
func (c *CatalogClient) ListResources(ctx context.Context, filters *ResourceFilters) ([]*models.ResourceDefinition, error)
func (c *CatalogClient) Subscribe(ctx context.Context, filters *EventFilters) (<-chan *Event, error)

// orchestrator/pkg/client/client.go
type OrchestratorClient struct {
    endpoint string
    auth     Authenticator
    timeout  time.Duration
}

func NewOrchestratorClient(config *ClientConfig) *OrchestratorClient

func (c *OrchestratorClient) CreateDeployment(ctx context.Context, req *DeploymentRequest) (*Deployment, error)
func (c *OrchestratorClient) GetDeployment(ctx context.Context, id string) (*Deployment, error)
func (c *OrchestratorClient) WatchDeploymentStatus(ctx context.Context, id string) (<-chan *DeploymentStatus, error)

// composer/pkg/client/client.go
type ComposerClient struct {
    endpoint string
    auth     Authenticator
    team     string
    timeout  time.Duration
}

func NewComposerClient(config *ClientConfig) *ComposerClient

func (c *ComposerClient) CreateComposition(ctx context.Context, comp *ComposedResource) error
func (c *ComposerClient) Deploy(ctx context.Context, req *TeamDeploymentRequest) (*TeamDeployment, error)
func (c *ComposerClient) ListDeployments(ctx context.Context, filters *DeploymentFilters) ([]*TeamDeployment, error)
```

## 🎯 API Versioning Strategy

### **Version Management**
```go
// All services support API versioning
type APIVersion struct {
    Version    string `json:"version"`    // "v1", "v2alpha1", "v1beta1"
    Deprecated bool   `json:"deprecated"`
    Sunset     *time.Time `json:"sunset,omitempty"`
}

// Version negotiation
GET /api/versions              # List supported API versions
GET /api/v1/...               # Version-specific endpoints
GET /api/v2beta1/...          # Beta version endpoints
```

### **Backward Compatibility**
```go
// Interface evolution patterns
type ResourceDefinitionV1 struct {
    Metadata ResourceMetadataV1 `json:"metadata"`
    Spec     ResourceSpecV1     `json:"spec"`
}

type ResourceDefinitionV2 struct {
    Metadata ResourceMetadataV2 `json:"metadata"`
    Spec     ResourceSpecV2     `json:"spec"`
    Status   ResourceStatus     `json:"status"` // New field in v2
}

// Migration functions
func ConvertV1ToV2(v1 *ResourceDefinitionV1) *ResourceDefinitionV2
func ConvertV2ToV1(v2 *ResourceDefinitionV2) *ResourceDefinitionV1
```

---

**These interfaces provide the foundation for a scalable, maintainable platform architecture. All services implement these contracts consistently, enabling easy testing, evolution, and integration.**
