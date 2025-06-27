// This file documents the key interfaces designed for the Nestor Orchestrator
// Keep this in sync with actual implementations

// =============================================================================
// STORAGE INTERFACES (internal/storage/interfaces.go)
// =============================================================================

// CatalogStore - Main storage interface combining all component operations
type CatalogStore interface {
	ComponentReader
	ComponentWriter
	ComponentSearcher
	ComponentVersioning
	ComponentValidator
}

// ComponentReader - Read operations for component definitions
type ComponentReader interface {
	GetComponent(ctx context.Context, name, version string) (*models.ComponentDefinition, error)
	GetLatestComponent(ctx context.Context, name string) (*models.ComponentDefinition, error)
	ListComponents(ctx context.Context, req *ListComponentsRequest) (*ListComponentsResponse, error)
	GetComponentVersions(ctx context.Context, name string) ([]*models.ComponentVersion, error)
	ComponentExists(ctx context.Context, name, version string) (bool, error)
}

// ComponentWriter - Write operations with validation-before-write pattern
type ComponentWriter interface {
	CreateComponent(ctx context.Context, component *models.ComponentDefinition) error
	UpdateComponent(ctx context.Context, component *models.ComponentDefinition) error
	DeleteComponent(ctx context.Context, name string, reason string) error
	DeleteComponentVersion(ctx context.Context, name, version string, reason string) error
	BatchCreateComponents(ctx context.Context, components []*models.ComponentDefinition) error
}

// ComponentSearcher - Discovery and filtering operations
type ComponentSearcher interface {
	SearchComponents(ctx context.Context, req *SearchComponentsRequest) (*SearchComponentsResponse, error)
	FindComponentsByProvider(ctx context.Context, provider string) ([]*models.ComponentDefinition, error)
	FindComponentsByCategory(ctx context.Context, category string) ([]*models.ComponentDefinition, error)
	FindComponentsByLabels(ctx context.Context, labels map[string]string) ([]*models.ComponentDefinition, error)
	FindDependents(ctx context.Context, componentName string) ([]*models.ComponentDefinition, error)
	FindDependencies(ctx context.Context, componentName, version string, recursive bool) ([]*models.ComponentDefinition, error)
}

// ComponentVersioning - Semantic versioning and change tracking
type ComponentVersioning interface {
	CreateComponentVersion(ctx context.Context, version *models.ComponentVersion) error
	GetComponentVersion(ctx context.Context, name, version string) (*models.ComponentVersion, error)
	GetLatestMajorVersion(ctx context.Context, name string, majorVersion int) (*models.ComponentVersion, error)
	ListComponentChanges(ctx context.Context, name string, since time.Time) ([]*models.ComponentChange, error)
	CompareVersions(ctx context.Context, name, fromVersion, toVersion string) (*models.VersionDiff, error)
	GetVersionsByCommit(ctx context.Context, commitSHA string) ([]*models.ComponentVersion, error)
	FindCompatibleVersions(ctx context.Context, name, constraint string) ([]*models.ComponentVersion, error)
}

// =============================================================================
// VALIDATION INTERFACES (internal/validation/interfaces.go)
// =============================================================================

// ComponentValidator - Business logic validation with caching
type ComponentValidator interface {
	ValidateSemanticVersioning(ctx context.Context, name, currentVersion, newVersion string, changes *models.VersionDiff) (*ValidationResult, error)
	ValidateDependencies(ctx context.Context, component *models.ComponentDefinition) (*ValidationResult, error)
	ValidateCircularDependencies(ctx context.Context, component *models.ComponentDefinition) (*ValidationResult, error)
	ValidateBreakingChanges(ctx context.Context, current, updated *models.ComponentDefinition) (*ValidationResult, error)
	ValidateDeploymentEngines(ctx context.Context, component *models.ComponentDefinition) (*ValidationResult, error)
	ValidateConflicts(ctx context.Context, component *models.ComponentDefinition) (*ValidationResult, error)
	ValidateInputOutputSpecs(ctx context.Context, component *models.ComponentDefinition) (*ValidationResult, error)

	// Cache management
	GetCachedValidation(ctx context.Context, componentName, version string) (*ValidationResult, error)
	CacheValidationResult(ctx context.Context, componentName, version string, result *ValidationResult) error
	InvalidateValidationCache(ctx context.Context, componentName string) error
}

// =============================================================================
// ENGINE INTERFACES (internal/engines/interfaces.go)
// =============================================================================

// DeploymentEngineRegistry - Manages available deployment engines
type DeploymentEngineRegistry interface {
	RegisterEngine(ctx context.Context, engine *DeploymentEngine) error
	GetEngine(ctx context.Context, name string) (*DeploymentEngine, error)
	ListEngines(ctx context.Context) ([]*DeploymentEngine, error)
	IsEngineHealthy(ctx context.Context, name string) (bool, error)
	GetEngineCapabilities(ctx context.Context, name string) (*EngineCapabilities, error)
}

// =============================================================================
// KEY DATA STRUCTURES
// =============================================================================

// Component metadata structure
type ComponentMetadata struct {
	Name         string            `json:"name"`
	DisplayName  string            `json:"display_name"`
	Description  string            `json:"description"`
	Version      string            `json:"version"`      // semantic version

	Provider     string            `json:"provider"`      // aws, gcp, azure, k8s
	Category     string            `json:"category"`      // database, compute, storage
	SubCategory  string            `json:"sub_category"`  // rds, ec2, s3, vpc
	ResourceType string            `json:"resource_type"` // mysql, postgresql, redis

	DeploymentEngines []string      `json:"deployment_engines"`
	RequiredEngines   []string      `json:"required_engines"`

	Dependencies      []Dependency  `json:"dependencies"`
	Provides          []string      `json:"provides"`
	ConflictsWith     []string      `json:"conflicts_with"`

	RequiredInputs    []InputSpec   `json:"required_inputs"`
	OptionalInputs    []InputSpec   `json:"optional_inputs"`
	Outputs          []OutputSpec   `json:"outputs"`

	Maturity         MaturityLevel  `json:"maturity"`
	SupportLevel     SupportLevel   `json:"support_level"`
	Maintainers      []string       `json:"maintainers"`
	Documentation    []DocLink      `json:"documentation"`

	ResourceLimits   ResourceLimits `json:"resource_limits"`
	CostEstimate     CostEstimate   `json:"cost_estimate"`

	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeprecatedAt     *time.Time     `json:"deprecated_at,omitempty"`

	GitRepository    string         `json:"git_repository"`
	GitPath          string         `json:"git_path"`
	GitCommit        string         `json:"git_commit"`
	GitBranch        string         `json:"git_branch"`

	Labels           map[string]string `json:"labels"`
	Annotations      map[string]string `json:"annotations"`
}

// Validation result structure
type ValidationResult struct {
	Valid              bool                  `json:"valid"`
	Errors            []ValidationError     `json:"errors"`
	Warnings          []ValidationWarning   `json:"warnings"`
	Summary           ValidationSummary     `json:"summary"`
	SuggestedVersion  *string               `json:"suggested_version,omitempty"`
}

// Deployment engine structure
type DeploymentEngine struct {
	Name         string            `json:"name"`
	DisplayName  string            `json:"display_name"`
	Version      string            `json:"version"`
	Endpoint     string            `json:"endpoint"`
	Status       EngineStatus      `json:"status"`
	Capabilities EngineCapabilities `json:"capabilities"`
	LastChecked  time.Time         `json:"last_checked"`
	Metadata     map[string]string `json:"metadata"`
}

// =============================================================================
// CONFIGURATION PATTERNS
// =============================================================================

// Each package should have a Config struct for its dependencies
type CatalogStoreConfig struct {
	TableName     string
	Region        string
	Endpoint      string

	CacheEnabled  bool
	CacheTTL      time.Duration

	ValidationCacheEnabled bool
	ValidationCacheTTL     time.Duration
	ValidationCacheSize    int

	BatchSize     int
	MaxRetries    int
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration

	EnableFullTextSearch bool
	SearchIndexName      string

	EngineRegistryEnabled   bool
	EngineHealthCheckPeriod time.Duration
}

// =============================================================================
// ERROR HANDLING PATTERNS
// =============================================================================

// All packages use centralized error handling from pkg/errors
// Key error codes for component operations:

const (
	// Component errors
	ErrorCodeComponentNotFound       ErrorCode = "COMPONENT_NOT_FOUND"
	ErrorCodeComponentAlreadyExists  ErrorCode = "COMPONENT_ALREADY_EXISTS"
	ErrorCodeInvalidVersion          ErrorCode = "INVALID_VERSION"
	ErrorCodeVersionConflict         ErrorCode = "VERSION_CONFLICT"

	// Validation errors
	ErrorCodeValidationFailed        ErrorCode = "VALIDATION_FAILED"
	ErrorCodeDependencyError         ErrorCode = "DEPENDENCY_ERROR"
	ErrorCodeBreakingChange          ErrorCode = "BREAKING_CHANGE"
	ErrorCodeCircularDependency      ErrorCode = "CIRCULAR_DEPENDENCY"

	// Engine errors
	ErrorCodeEngineNotFound          ErrorCode = "ENGINE_NOT_FOUND"
	ErrorCodeEngineUnhealthy         ErrorCode = "ENGINE_UNHEALTHY"
	ErrorCodeEngineUnsupported       ErrorCode = "ENGINE_UNSUPPORTED"

	// Storage errors
	ErrorCodeStorageFailure          ErrorCode = "STORAGE_FAILURE"
	ErrorCodeCacheFailure            ErrorCode = "CACHE_FAILURE"
	ErrorCodeDatabaseConnection      ErrorCode = "DATABASE_CONNECTION"
)

// =============================================================================
// CONSTRUCTOR PATTERNS
// =============================================================================

// All constructors follow this pattern:
func NewCatalogManager(
	store storage.CatalogStore,
	validator validation.ComponentValidator,
	logger logging.Logger,
) *CatalogManager

func NewDynamoDBCatalogStore(
	client *dynamodb.Client,
	cache cache.Client,
	logger logging.Logger,
) storage.CatalogStore

func NewComponentValidator(
	store storage.CatalogStore,
	engineRegistry engines.DeploymentEngineRegistry,
	cache cache.Client,
	logger logging.Logger,
) validation.ComponentValidator

// =============================================================================
// TESTING PATTERNS
// =============================================================================

// Use in-memory implementations for testing
func NewInMemoryCatalogStore() storage.CatalogStore
func NewMockValidator() validation.ComponentValidator

// Test example:
func TestCatalogManager_CreateComponent(t *testing.T) {
	store := memory.NewCatalogStore()
	validator := &mockValidator{}
	logger := logging.NewNoop()

	manager := catalog.NewManager(store, validator, logger)

	err := manager.CreateComponent(ctx, component)
	assert.NoError(t, err)
}
