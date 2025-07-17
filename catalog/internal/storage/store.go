// catalog/internal/storage/store.go
package storage

import (
	"context"
	"time"

	"github.com/HatiCode/nestor/catalog/pkg/models"
)

// ComponentStore is the main interface for component storage operations
// This interface aligns with the design document requirements
type ComponentStore interface {
	GetComponent(ctx context.Context, name, version string) (*models.ComponentDefinition, error)
	ListComponents(ctx context.Context, filters ComponentFilters, pagination Pagination) (*ComponentList, error)
	StoreComponent(ctx context.Context, component *models.ComponentDefinition) error
	GetVersionHistory(ctx context.Context, name string) ([]models.ComponentVersion, error)
	HealthCheck(ctx context.Context) error
}

// Types for ComponentStore interface
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

type Pagination struct {
	Limit     int32     `json:"limit" validate:"min=1,max=100"`
	NextToken string    `json:"next_token"`
	SortBy    SortField `json:"sort_by"`
	SortOrder SortOrder `json:"sort_order"`
}

type ComponentList struct {
	Components []*models.ComponentDefinition `json:"components"`
	NextToken  string                        `json:"next_token,omitempty"`
	Total      int64                         `json:"total"`
	HasMore    bool                          `json:"has_more"`
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

// Validation methods
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

func (p *Pagination) Validate() error {
	if p.Limit <= 0 {
		p.Limit = 20
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	return nil
}

// Additional error definitions that might be needed in the future
// can be added here as the implementation evolves

// Error definitions
var (
	ErrComponentNotFound      = NewStorageError("COMPONENT_NOT_FOUND", "component not found")
	ErrVersionNotFound        = NewStorageError("VERSION_NOT_FOUND", "component version not found")
	ErrComponentExists        = NewStorageError("COMPONENT_EXISTS", "component already exists")
	ErrInvalidVersion         = NewStorageError("INVALID_VERSION", "invalid semantic version")
	ErrInvalidDateRange       = NewStorageError("INVALID_DATE_RANGE", "start date must be before end date")
	ErrEmptySearchQuery       = NewStorageError("EMPTY_SEARCH_QUERY", "search query cannot be empty")
	ErrUnsupportedStorageType = NewStorageError("UNSUPPORTED_STORAGE_TYPE", "unsupported storage type")
	ErrStorageNotAvailable    = NewStorageError("STORAGE_NOT_AVAILABLE", "storage backend not available")
	ErrInvalidInput           = NewStorageError("INVALID_INPUT", "invalid input provided")
	ErrInvalidConfig          = NewStorageError("INVALID_CONFIG", "invalid storage configuration")
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
