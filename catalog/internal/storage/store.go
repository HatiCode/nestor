// catalog/internal/storage/store.go
package storage

import (
	"context"
	"time"

	"github.com/HatiCode/nestor/catalog/pkg/models"
)

// ComponentStore is the main interface for component storage operations
type ComponentStore interface {
	GetComponent(ctx context.Context, name, version string) (*models.ComponentDefinition, error)
	ListComponents(ctx context.Context, filters ComponentFilters, pagination Pagination) (*ComponentList, error)
	StoreComponent(ctx context.Context, component *models.ComponentDefinition) error
	GetVersionHistory(ctx context.Context, name string) ([]models.ComponentVersion, error)
	HealthCheck(ctx context.Context) error
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
