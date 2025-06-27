# Nestor Orchestrator - Architecture & Development Guide

This document defines the architectural decisions, coding philosophy, and project structure for the Nestor Orchestrator component. **Read this first** before making any code changes to ensure consistency with the established patterns.

## 🏗️ Architecture Overview

The Orchestrator is the central coordination hub that manages a **global component catalog** (the "buffet") containing base infrastructure components. These components are synced to processors where teams compose them into complex deployments.

### Core Principles

1. **Global Component Catalog** - No team isolation at the storage level; all components are globally accessible
2. **Interface-Driven Design** - All external dependencies are behind interfaces for testability and flexibility
3. **Cache-Transparent Storage** - Caching is handled internally by implementations, not exposed in interfaces
4. **Validation Before Write** - All validation happens before storage operations, with caching of validation results
5. **Direct Dependencies Only** - Dependency validation focuses on direct dependencies for performance
6. **Semantic Versioning** - Components use strict semver with automatic breaking change detection
7. **Real-time Updates** - Server-Sent Events (SSE) for real-time component synchronization to processors

## 📁 Project Structure

```
orchestrator/
├── main.go                           # Application entry point with natural DI
├── go.mod                           # Module definition
├── README.md                        # Component documentation
├── CHANGELOG.md                     # Version history
│
├── cmd/                             # Command line interface
│   ├── root.go                      # Root command setup
│   ├── serve.go                     # HTTP server command
│   ├── migrate.go                   # Database migration command
│   └── version.go                   # Version command
│
├── internal/                        # Private implementation (not importable)
│   ├── config/                      # Configuration management
│   │   ├── config.go                # Configuration structs and loading
│   │   └── validation.go            # Config validation
│   │
│   ├── api/                         # HTTP API layer
│   │   ├── server.go                # HTTP server implementation
│   │   ├── middleware/              # HTTP middleware
│   │   └── handlers/                # HTTP request handlers
│   │       ├── catalog.go           # Component catalog endpoints
│   │       ├── health.go            # Health check endpoints
│   │       ├── sse.go               # Server-sent events
│   │       └── version.go           # Version endpoint
│   │
│   ├── catalog/                     # Catalog business logic
│   │   ├── manager.go               # Core catalog management
│   │   ├── sync.go                  # Git synchronization
│   │   └── events.go                # Catalog change events
│   │
│   ├── storage/                     # Storage layer
│   │   ├── interfaces.go            # Storage interfaces
│   │   ├── dynamodb/                # DynamoDB implementation
│   │   ├── memory/                  # In-memory implementation (testing)
│   │   └── cache/                   # Caching implementations
│   │
│   ├── validation/                  # Component validation
│   │   ├── interfaces.go            # Validation interfaces
│   │   ├── validator.go             # Main validator implementation
│   │   ├── rules/                   # Validation rule implementations
│   │   └── cache.go                 # Validation result caching
│   │
│   ├── engines/                     # Deployment engine management
│   │   ├── interfaces.go            # Engine interfaces
│   │   ├── registry.go              # Engine registry implementation
│   │   ├── health.go                # Engine health checking
│   │   └── discovery.go             # Engine discovery
│   │
│   ├── events/sse/                  # Server-sent events
│   │   ├── server.go                # SSE server implementation
│   │   ├── client.go                # SSE client management
│   │   └── events.go                # Event types and handlers
│   │
│   ├── git/                         # Git integration
│   │   ├── client.go                # Git client wrapper
│   │   ├── webhook.go               # Git webhook handlers
│   │   └── parser.go                # Component definition parser
│   │
│   └── observability/               # Metrics, logging, tracing
│       ├── metrics/                 # Prometheus metrics
│       ├── logging/                 # Structured logging
│       └── tracing/                 # Distributed tracing
│
├── pkg/                            # Public APIs (importable by processors)
│   ├── api/                        # Client libraries
│   │   ├── client.go               # HTTP API client
│   │   ├── sse_client.go           # SSE client for processors
│   │   └── types.go                # Request/response types
│   ├── models/                     # Data models
│   │   ├── component.go            # ComponentDefinition model
│   │   ├── version.go              # Version-related models
│   │   ├── validation.go           # ValidationResult models
│   │   └── engine.go               # DeploymentEngine models
│   ├── errors/                     # Centralized error handling
│   │   ├── errors.go               # Core error types and builders
│   │   ├── codes.go                # Error code definitions
│   │   ├── http.go                 # HTTP error mapping
│   │   └── validation.go           # Validation-specific errors
│   └── events/                     # Event definitions for SSE
│       ├── catalog.go              # Catalog change events
│       └── types.go                # Event type definitions
│
├── configs/                        # Configuration files
├── deployments/                    # K8s, Docker, Helm manifests
├── migrations/                     # Database migrations
├── examples/                       # Sample component definitions
├── test/                          # Test files and fixtures
└── docs/                          # Architecture documentation
```

## 🔧 Coding Philosophy

### Dependency Injection - Natural Go Patterns

**DO NOT** use dependency injection frameworks. Use Go's natural constructor pattern:

```go
// ✅ Good - Natural Go constructor pattern
func NewCatalogManager(
    store storage.CatalogStore,
    validator validation.ComponentValidator,
    logger logging.Logger,
) *CatalogManager {
    return &CatalogManager{
        store:     store,
        validator: validator,
        logger:    logger,
    }
}

// ✅ Good - Wire dependencies in main.go
func main() {
    logger := logging.New(config.Log)
    cache := cache.NewRedisClient(config.Cache, logger)
    store := dynamodb.NewCatalogStore(config.DB, cache, logger)
    validator := validation.NewValidator(store, logger)
    manager := catalog.NewManager(store, validator, logger)

    server := api.NewServer(manager, logger)
    server.Start()
}

// ❌ Bad - Don't use DI frameworks or containers
```

### Interface Design - Follow SOLID Principles

**Interface Segregation**: Split large interfaces into focused, single-responsibility interfaces:

```go
// ✅ Good - Small, focused interfaces
type ComponentReader interface {
    GetComponent(ctx context.Context, name, version string) (*ComponentDefinition, error)
    ListComponents(ctx context.Context, req *ListComponentsRequest) (*ListComponentsResponse, error)
}

type ComponentWriter interface {
    CreateComponent(ctx context.Context, component *ComponentDefinition) error
    UpdateComponent(ctx context.Context, component *ComponentDefinition) error
}

// Combine them in the main interface
type CatalogStore interface {
    ComponentReader
    ComponentWriter
    ComponentValidator
}

// ❌ Bad - Monolithic interfaces with unrelated methods
```

### Error Handling - Centralized and Structured

**Always use the centralized error package** for consistent error handling:

```go
// ✅ Good - Rich, structured errors
return errors.New(errors.ErrorCodeComponentNotFound).
    Component("catalog").
    Operation("GetComponent").
    Message("component not found").
    Detail("component_name", name).
    Detail("version", version).
    Build()

// ✅ Good - Wrap external errors
dbErr := db.GetItem(...)
if dbErr != nil {
    return errors.StorageFailure("catalog", "GetComponent", dbErr)
}

// ❌ Bad - Generic errors without context
return fmt.Errorf("component not found")
```

### Validation Strategy - Validate Before Write

**All write operations MUST validate before storage**:

```go
// ✅ Good - Validation before write with caching
func (c *catalogStore) CreateComponent(ctx context.Context, component *ComponentDefinition) error {
    // 1. Check validation cache
    if cached := c.validator.GetCachedValidation(ctx, component.Name, component.Version); cached != nil {
        if !cached.Valid {
            return errors.ValidationFailed("catalog", "CreateComponent", cached.Errors)
        }
    } else {
        // 2. Run validation
        result, err := c.validator.ValidateComponent(ctx, component)
        if err != nil {
            return err
        }

        // 3. Cache result
        c.validator.CacheValidationResult(ctx, component.Name, component.Version, result)

        if !result.Valid {
            return errors.ValidationFailed("catalog", "CreateComponent", result.Errors)
        }
    }

    // 4. Write only if validation passes
    return c.storage.CreateComponent(ctx, component)
}
```

### Caching Strategy - Transparent to Interfaces

**Caching should be handled internally by implementations**:

```go
// ✅ Good - Cache-transparent interface
type CatalogStore interface {
    GetComponent(ctx context.Context, name, version string) (*ComponentDefinition, error)
}

// ✅ Good - Implementation handles caching internally
func (s *dynamodbStore) GetComponent(ctx context.Context, name, version string) (*ComponentDefinition, error) {
    key := fmt.Sprintf("%s:%s", name, version)

    // Check cache first
    if cached := s.cache.Get(key); cached != nil {
        return cached.(*ComponentDefinition), nil
    }

    // Fallback to storage
    component, err := s.db.GetItem(...)
    if err == nil {
        s.cache.Set(key, component, 5*time.Minute)
    }
    return component, err
}

// ❌ Bad - Cache-aware interface
type CatalogStore interface {
    GetComponent(ctx context.Context, name, version string, opts *CacheOptions) (*ComponentDefinition, error)
}
```

## 🎯 Component Architecture Decisions

### Catalog Store Interface

The main storage interface is split into focused sub-interfaces:

- **ComponentReader**: Read operations (GetComponent, ListComponents, etc.)
- **ComponentWriter**: Write operations (CreateComponent, UpdateComponent, etc.)
- **ComponentSearcher**: Search and discovery operations
- **ComponentVersioning**: Version management and history
- **ComponentValidator**: Business logic validation

### Validation Rules

1. **Semantic Versioning**: Enforce semver rules and detect breaking changes
2. **Dependency Validation**: Check direct dependencies only (not transitive)
3. **Engine Validation**: Verify deployment engines exist and are healthy
4. **Conflict Detection**: Prevent naming and resource type conflicts
5. **Input/Output Validation**: Ensure component interfaces are well-defined

### Component Metadata

Components include rich metadata for discovery and validation:

```go
type ComponentMetadata struct {
    // Identity
    Name         string `json:"name"`
    Version      string `json:"version"`      // semantic version
    Description  string `json:"description"`

    // Classification
    Provider     string `json:"provider"`     // aws, gcp, azure, k8s
    Category     string `json:"category"`     // database, compute, storage
    ResourceType string `json:"resource_type"` // mysql, postgresql, redis

    // Deployment
    DeploymentEngines []string `json:"deployment_engines"` // crossplane, pulumi, terraform

    // Dependencies (direct only)
    Dependencies []Dependency `json:"dependencies"`

    // Configuration
    RequiredInputs []InputSpec `json:"required_inputs"`
    OptionalInputs []InputSpec `json:"optional_inputs"`
    Outputs        []OutputSpec `json:"outputs"`

    // Operational
    Maturity       MaturityLevel `json:"maturity"`        // alpha, beta, stable
    SupportLevel   SupportLevel  `json:"support_level"`   // community, supported
    ResourceLimits ResourceLimits `json:"resource_limits"` // cpu, memory estimates

    // Git source tracking
    GitRepository string `json:"git_repository"`
    GitCommit     string `json:"git_commit"`
    GitPath       string `json:"git_path"`

    // Flexible metadata
    Labels      map[string]string `json:"labels"`
    Annotations map[string]string `json:"annotations"`
}
```

## 🔍 Query Patterns

The storage layer is optimized for these primary query patterns:

1. **Component Lookup**: `GetComponent(name, version)` - Most frequent operation
2. **Latest Version**: `GetLatestComponent(name)` - Common for dependency resolution
3. **Component Discovery**: `ListComponents()` with filtering by provider/category
4. **Search**: `SearchComponents(query)` - Text search across names/descriptions
5. **Dependency Queries**: `FindDependencies(name, version)` - Direct dependencies only
6. **Version History**: `GetComponentVersions(name)` - All versions of a component

## 🚀 Development Workflow

### Adding New Features

1. **Define interfaces first** in the appropriate `interfaces.go` file
2. **Create models** in `pkg/models/` if needed for external consumption
3. **Implement business logic** in the internal packages
4. **Add validation rules** if the feature affects component validation
5. **Update error codes** in `pkg/errors/` if new error types are needed
6. **Write tests** with mocked dependencies
7. **Update API handlers** to expose the functionality

### Testing Strategy

- **Unit tests**: Mock all dependencies using interfaces
- **Integration tests**: Use in-memory implementations for storage
- **Component tests**: Test full request/response cycles
- **Contract tests**: Ensure interfaces are properly implemented

### File Naming Conventions

- `interfaces.go` - Interface definitions for each package
- `<domain>.go` - Main implementation (e.g., `catalog.go`, `validator.go`)
- `<domain>_test.go` - Unit tests
- `models.go` - Package-specific data structures
- `errors.go` - Package-specific error definitions (if needed)

## 🎯 Key Design Goals

1. **Scalability**: Read-heavy workload optimization with caching
2. **Reliability**: Comprehensive validation and error handling
3. **Maintainability**: Clean interfaces and separation of concerns
4. **Testability**: Mockable interfaces and dependency injection
5. **Observability**: Structured logging, metrics, and tracing throughout
6. **Performance**: Efficient query patterns and caching strategies

## 🚫 What NOT to Do

1. **Don't use DI frameworks** - Use natural Go constructor patterns
2. **Don't create monolithic interfaces** - Keep interfaces focused and small
3. **Don't validate transitive dependencies** - Only direct dependencies for performance
4. **Don't expose caching in interfaces** - Keep caching transparent to consumers
5. **Don't create team isolation** - The catalog is global to all teams
6. **Don't use schema validation** - Focus on business logic validation only
7. **Don't couple packages tightly** - Use interfaces for all cross-package dependencies

## 📚 References

- [SOLID Principles in Go](https://dave.cheney.net/2016/08/20/solid-go-design)
- [Go Project Structure](https://github.com/golang-standards/project-layout)
- [Semantic Versioning](https://semver.org/)
- [Effective Go](https://golang.org/doc/effective_go.html)

---

**This document should be updated whenever architectural decisions change. All team members should be familiar with these patterns before contributing to the codebase.**
