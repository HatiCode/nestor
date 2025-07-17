# Scalable Storage Architecture

## 📁 Directory Structure

```
internal/storage/
├── store.go            # Main ComponentStore interface (database-agnostic)
├── cache.go            # Cache interface
├── config.go           # Configuration types
├── factory.go          # Factory for creating store implementations
│
├── dynamodb/           # DynamoDB implementation
│   ├── component_store.go # ComponentStore implementation
│   ├── client.go       # DynamoDB client wrapper
│   ├── config.go       # DynamoDB configuration
│   ├── init.go         # Registration with factory
│   └── models.go       # DynamoDB-specific models
│
├── postgres/           # PostgreSQL implementation (future)
│   ├── component_store.go # ComponentStore implementation
│   ├── client.go       # PostgreSQL client wrapper
│   ├── queries.go      # SQL query builders
│   ├── migrations.go   # Database migrations
│   └── models.go       # PostgreSQL-specific models
│
├── memory/             # In-memory implementation (testing)
│   └── component_store.go # ComponentStore implementation
│
internal/catalog/       # Business logic (separate from storage)
├── manager.go          # Catalog management business logic
├── sync.go             # Git synchronization
└── events.go           # Catalog change events
```

## 🏗️ Clear Separation of Concerns

### Storage Layer (`internal/storage/`)
- **Pure data persistence** - CRUD operations, queries, caching
- **Database-agnostic interfaces** - can swap implementations
- **No business logic** - just storage and retrieval

### Business Logic Layer (`internal/catalog/`)
- **Catalog management** - component lifecycle, validation coordination
- **Git synchronization** - parsing components from Git repos
- **Event publishing** - SSE notifications to processors
- **Uses storage layer** - but doesn't know about specific databases

## 🎯 Implementation Pattern

### Main Interface (Database-Agnostic)
```go
// internal/storage/store.go
type ComponentStore interface {
    GetComponent(ctx context.Context, name, version string) (*models.ComponentDefinition, error)
    ListComponents(ctx context.Context, filters ComponentFilters, pagination Pagination) (*ComponentList, error)
    StoreComponent(ctx context.Context, component *models.ComponentDefinition) error
    GetVersionHistory(ctx context.Context, name string) ([]models.ComponentVersion, error)
    HealthCheck(ctx context.Context) error
}
```

### Business Logic Layer
```go
// internal/catalog/manager.go
package catalog

import "github.com/HatiCode/nestor/catalog/internal/storage"

type Manager struct {
    store     storage.ComponentStore  // Uses storage interface
    validator ComponentValidator      // Local interface
    logger    Logger
}

func NewManager(store storage.ComponentStore, validator ComponentValidator, logger Logger) *Manager {
    return &Manager{
        store:     store,
        validator: validator,
        logger:    logger,
    }
}

// Business logic methods
func (m *Manager) PublishComponent(ctx context.Context, component *models.ComponentDefinition) error {
    // 1. Validate component
    if err := m.validator.ValidateComponent(ctx, component); err != nil {
        return err
    }

    // 2. Store component
    if err := m.store.StoreComponent(ctx, component); err != nil {
        return err
    }

    // 3. Publish event (business logic)
    m.publishComponentEvent(component)

    return nil
}
```

### DynamoDB Implementation
```go
// internal/storage/dynamodb/component_store.go
package dynamodb

import "github.com/HatiCode/nestor/catalog/internal/storage"

type componentStore struct {
    client    *Client
    cache     cache.Cache
    logger    logging.Logger
    tableName string
    config    *Config
}

// NewComponentStore creates a DynamoDB-backed ComponentStore
func NewComponentStore(config *storage.StorageConfig, cache cache.Cache, logger logging.Logger) (storage.ComponentStore, error) {
    // Convert storage config to DynamoDB config
    dynamoConfig, err := convertStorageConfig(config.DynamoDB)
    if err != nil {
        return nil, fmt.Errorf("failed to convert storage config: %w", err)
    }
    
    client, err := NewClient(dynamoConfig, logger)
    if err != nil {
        return nil, fmt.Errorf("failed to create DynamoDB client: %w", err)
    }
    
    store := &componentStore{
        client:    client,
        cache:     cache,
        logger:    logger.With("component", "dynamodb_component_store"),
        tableName: dynamoConfig.GetTableName(),
        config:    dynamoConfig,
    }
    
    return store, nil
}

// Pure storage implementation - no business logic
func (s *componentStore) GetComponent(ctx context.Context, name, version string) (*models.ComponentDefinition, error) {
    // 1. Check cache first
    cacheKey := s.buildComponentCacheKey(name, version)
    if s.cache != nil {
        if cached := s.cache.Get(ctx, cacheKey); cached != nil {
            if component, ok := cached.(*models.ComponentDefinition); ok {
                return component, nil
            }
        }
    }

    // 2. Query DynamoDB
    result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
        TableName: aws.String(s.tableName),
        Key: map[string]types.AttributeValue{
            "PK": &types.AttributeValueMemberS{Value: s.buildComponentPK(name)},
            "SK": &types.AttributeValueMemberS{Value: s.buildVersionSK(version)},
        },
    })
    
    // 3. Parse and cache result
    // ... implementation details
}
```

## 🔧 Usage in Main Application

```go
// main.go - Clean dependency injection
func main() {
    config := loadConfig()
    logger := logging.New(config.Logging)
    cache := cache.NewRedisClient(config.Cache, logger)

    // Factory creates the right implementation based on config
    componentStore, err := storage.NewComponentStore(config.Storage, cache, logger)
    if err != nil {
        log.Fatal(err)
    }

    // Rest of the app doesn't know/care about the implementation
    catalogManager := catalog.NewManager(componentStore, validator, logger)
    server := api.NewServer(catalogManager, logger)
    server.Start()
}
```

## 🎯 Benefits of This Architecture

### ✅ **Database Agnostic**
- Easy to switch between DynamoDB, PostgreSQL, MySQL, etc.
- Can even run multiple storage backends simultaneously

### ✅ **Idiomatic Go**
- Interfaces near usage (main interface in storage package)
- Implementation-specific code in separate packages
- No giant `interfaces.go` file

### ✅ **Testable**
- In-memory implementation for fast tests
- Each implementation can be tested independently
- Easy to mock the main interface

### ✅ **Scalable**
- Add new storage backends without changing existing code
- Performance characteristics can vary by implementation
- Different teams can work on different implementations

## 🚀 Migration Path

### Phase 1: DynamoDB (Current)
```yaml
storage:
  type: dynamodb
  dynamodb:
    table_name: nestor-components
    region: us-west-2
```

### Phase 2: Add PostgreSQL Option
```yaml
storage:
  type: postgres
  postgres:
    host: localhost
    database: nestor
    user: nestor_user
```

### Phase 3: Hybrid/Multi-Storage
```yaml
storage:
  type: hybrid
  primary: dynamodb    # writes go here
  replicas:           # reads can come from here
    - postgres
    - redis
```

## 📊 Implementation Priority

1. **DynamoDB** (immediate) - Your preferred choice
2. **Memory** (testing) - For unit tests and development
3. **PostgreSQL** (future) - For customers preferring SQL
4. **Redis** (future) - For high-performance read replicas
5. **MySQL** (future) - For customers with existing MySQL infrastructure

This gives you the scalability you need while keeping the architecture clean and Go-idiomatic!
