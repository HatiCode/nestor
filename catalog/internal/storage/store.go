// catalog/internal/storage/store.go
package storage

import (
	"context"
	"time"

	"github.com/HatiCode/nestor/catalog/pkg/models"
)

type CatalogStore interface {
	ComponentReader
	ComponentWriter
	ComponentSearcher
	ComponentVersioning
	HealthChecker
}

type ComponentReader interface {
	GetComponent(ctx context.Context, name, version string) (*models.ComponentDefinition, error)
	GetLatestComponent(ctx context.Context, name string) (*models.ComponentDefinition, error)
	ComponentExists(ctx context.Context, name, version string) (bool, error)
	ListComponents(ctx context.Context, req *ListComponentsRequest) (*ListComponentsResponse, error)
	BatchGetComponents(ctx context.Context, refs []ComponentRef) ([]*models.ComponentDefinition, error)
}

type ComponentWriter interface {
	CreateComponent(ctx context.Context, component *models.ComponentDefinition) error
	UpdateComponent(ctx context.Context, component *models.ComponentDefinition) error
	DeleteComponent(ctx context.Context, name, version string) error
	BatchCreateComponents(ctx context.Context, components []*models.ComponentDefinition) error
}

type ComponentSearcher interface {
	SearchComponents(ctx context.Context, req *SearchComponentsRequest) (*SearchComponentsResponse, error)
	FindComponentsByProvider(ctx context.Context, provider string) ([]*models.ComponentDefinition, error)
	FindComponentsByCategory(ctx context.Context, category string) ([]*models.ComponentDefinition, error)
	FindComponentsByLabels(ctx context.Context, labels map[string]string) ([]*models.ComponentDefinition, error)
	FindDependents(ctx context.Context, componentName string) ([]*models.ComponentDefinition, error)
	FindDependencies(ctx context.Context, componentName, version string, recursive bool) ([]*models.ComponentDefinition, error)
}

type ComponentVersioning interface {
	CreateComponentVersion(ctx context.Context, version *models.ComponentVersion) error
	GetComponentVersion(ctx context.Context, name, version string) (*models.ComponentVersion, error)
	GetComponentVersions(ctx context.Context, name string) ([]*models.ComponentVersion, error)
	GetLatestMajorVersion(ctx context.Context, name string, majorVersion int) (*models.ComponentVersion, error)
	ListComponentChanges(ctx context.Context, name string, since time.Time) ([]*models.ComponentChange, error)
	CompareVersions(ctx context.Context, name, fromVersion, toVersion string) (*models.VersionDiff, error)
	FindCompatibleVersions(ctx context.Context, name, constraint string) ([]*models.ComponentVersion, error)
	GetVersionsByCommit(ctx context.Context, commitSHA string) ([]*models.ComponentVersion, error)
}

type HealthChecker interface {
	HealthCheck(ctx context.Context) error
	Ping(ctx context.Context) error
	GetStats(ctx context.Context) (*StorageStats, error)
}

type ComponentRef struct {
	Name    string `json:"name" validate:"required,dns1123"`
	Version string `json:"version" validate:"required,semver"`
}

type ListComponentsRequest struct {
	Limit     int32             `json:"limit" validate:"min=1,max=100"`
	NextToken string            `json:"next_token"`
	SortBy    SortField         `json:"sort_by"`
	SortOrder SortOrder         `json:"sort_order"`
	Filters   *ComponentFilters `json:"filters"`
}

type ListComponentsResponse struct {
	Components []*models.ComponentDefinition `json:"components"`
	NextToken  string                        `json:"next_token,omitempty"`
	Total      int64                         `json:"total"`
	HasMore    bool                          `json:"has_more"`
}

type SearchComponentsRequest struct {
	Query     string            `json:"query" validate:"required,min=1"`
	Filters   *ComponentFilters `json:"filters"`
	Limit     int32             `json:"limit" validate:"min=1,max=100"`
	NextToken string            `json:"next_token"`
	Facets    []string          `json:"facets"`
	Highlight bool              `json:"highlight"`
}

type SearchComponentsResponse struct {
	Components []*models.ComponentDefinition `json:"components"`
	NextToken  string                        `json:"next_token,omitempty"`
	Total      int64                         `json:"total"`
	HasMore    bool                          `json:"has_more"`
	Facets     map[string]*FacetResult       `json:"facets,omitempty"`
	QueryTime  time.Duration                 `json:"query_time"`
}

type ComponentFilters struct {
	Providers          []string               `json:"providers" validate:"dive,required"`
	Categories         []string               `json:"categories" validate:"dive,required"`
	SubCategories      []string               `json:"sub_categories" validate:"dive,required"`
	Labels             map[string]string      `json:"labels"`
	CreatedAfter       *time.Time             `json:"created_after"`
	CreatedBefore      *time.Time             `json:"created_before"`
	UpdatedAfter       *time.Time             `json:"updated_after"`
	UpdatedBefore      *time.Time             `json:"updated_before"`
	Maturity           []models.MaturityLevel `json:"maturity" validate:"dive,oneof=alpha beta stable deprecated"`
	ActiveOnly         bool                   `json:"active_only"`
	VersionConstraint  string                 `json:"version_constraint" validate:"omitempty,semver_constraint"`
	MajorVersion       *int                   `json:"major_version" validate:"omitempty,min=0"`
	DeploymentEngines  []string               `json:"deployment_engines" validate:"dive,required"`
	HasDependency      string                 `json:"has_dependency" validate:"omitempty,dns1123"`
	ProvidesDependency string                 `json:"provides_dependency" validate:"omitempty,dns1123"`
}

type FacetResult struct {
	Values []FacetValue `json:"values"`
	Total  int64        `json:"total"`
}

type FacetValue struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

type StorageStats struct {
	TotalComponents      int64     `json:"total_components"`
	TotalVersions        int64     `json:"total_versions"`
	ActiveComponents     int64     `json:"active_components"`
	DeprecatedComponents int64     `json:"deprecated_components"`
	ProvidersCount       int64     `json:"providers_count"`
	CategoriesCount      int64     `json:"categories_count"`
	LastUpdated          time.Time `json:"last_updated"`
	StorageSize          int64     `json:"storage_size_bytes,omitempty"`
}

type SortField string

const (
	SortByName       SortField = "name"
	SortByCreated    SortField = "created_at"
	SortByUpdated    SortField = "updated_at"
	SortByProvider   SortField = "provider"
	SortByCategory   SortField = "category"
	SortByVersion    SortField = "version"
	SortByMaturity   SortField = "maturity"
	SortByPopularity SortField = "popularity"
	SortByUsageCount SortField = "usage_count"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

func (r *ListComponentsRequest) Validate() error {
	if r.Limit <= 0 {
		r.Limit = 20
	}
	if r.Limit > 100 {
		r.Limit = 100
	}

	if r.Filters != nil {
		return r.Filters.Validate()
	}

	return nil
}

func (r *SearchComponentsRequest) Validate() error {
	if r.Query == "" {
		return ErrEmptySearchQuery
	}

	if r.Limit <= 0 {
		r.Limit = 20
	}
	if r.Limit > 100 {
		r.Limit = 100
	}

	if r.Filters != nil {
		return r.Filters.Validate()
	}

	return nil
}

func (f *ComponentFilters) Validate() error {
	if f == nil {
		return nil
	}

	if f.CreatedAfter != nil && f.CreatedBefore != nil && f.CreatedAfter.After(*f.CreatedBefore) {
		return ErrInvalidDateRange
	}

	if f.UpdatedAfter != nil && f.UpdatedBefore != nil && f.UpdatedAfter.After(*f.UpdatedBefore) {
		return ErrInvalidDateRange
	}

	return nil
}

var (
	ErrComponentNotFound      = NewStorageError("COMPONENT_NOT_FOUND", "component not found")
	ErrVersionNotFound        = NewStorageError("VERSION_NOT_FOUND", "component version not found")
	ErrComponentExists        = NewStorageError("COMPONENT_EXISTS", "component already exists")
	ErrInvalidVersion         = NewStorageError("INVALID_VERSION", "invalid semantic version")
	ErrInvalidDateRange       = NewStorageError("INVALID_DATE_RANGE", "start date must be before end date")
	ErrEmptySearchQuery       = NewStorageError("EMPTY_SEARCH_QUERY", "search query cannot be empty")
	ErrUnsupportedStorageType = NewStorageError("UNSUPPORTED_STORAGE_TYPE", "unsupported storage type")
	ErrStorageNotAvailable    = NewStorageError("STORAGE_NOT_AVAILABLE", "storage backend not available")
)

type StorageError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *StorageError) Error() string {
	return e.Message
}

func NewStorageError(code, message string) *StorageError {
	return &StorageError{
		Code:    code,
		Message: message,
		Details: make(map[string]any),
	}
}

func (e *StorageError) WithDetail(key string, value any) *StorageError {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}
