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
	ListComponents(ctx context.Context, req *ListComponentsRequest) (*ListComponentsResponse, error)
	GetComponentVersions(ctx context.Context, name string) ([]*models.ComponentVersion, error)
	ComponentExists(ctx context.Context, name, version string) (bool, error)
}

type ComponentWriter interface {
	CreateComponent(ctx context.Context, component *models.ComponentDefinition) error
	UpdateComponent(ctx context.Context, component *models.ComponentDefinition) error
	DeleteComponent(ctx context.Context, name string, reason string) error
	DeleteComponentVersion(ctx context.Context, name, version string, reason string) error
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
	GetLatestMajorVersion(ctx context.Context, name string, majorVersion int) (*models.ComponentVersion, error)
	ListComponentChanges(ctx context.Context, name string, since time.Time) ([]*models.ComponentChange, error)
	CompareVersions(ctx context.Context, name, fromVersion, toVersion string) (*models.VersionDiff, error)
	GetVersionsByCommit(ctx context.Context, commitSHA string) ([]*models.ComponentVersion, error)
	FindCompatibleVersions(ctx context.Context, name, constraint string) ([]*models.ComponentVersion, error)
}

type HealthChecker interface {
	HealthCheck(ctx context.Context) error
	Ping(ctx context.Context) error
}

type ListComponentsRequest struct {
	Limit     int32             `json:"limit"`
	NextToken string            `json:"next_token"`
	SortBy    SortField         `json:"sort_by"`
	SortOrder SortOrder         `json:"sort_order"`
	Filters   *ComponentFilters `json:"filters"`
}

type ListComponentsResponse struct {
	Components []*models.ComponentDefinition `json:"components"`
	NextToken  string                        `json:"next_token"`
	Total      int64                         `json:"total"`
}

type SearchComponentsRequest struct {
	Query     string            `json:"query"`
	Filters   *ComponentFilters `json:"filters"`
	Limit     int32             `json:"limit"`
	NextToken string            `json:"next_token"`
	Facets    []string          `json:"facets"`
}

type SearchComponentsResponse struct {
	Components []*models.ComponentDefinition `json:"components"`
	NextToken  string                        `json:"next_token"`
	Total      int64                         `json:"total"`
	Facets     map[string]*FacetResult       `json:"facets"`
}

type ComponentFilters struct {
	Providers          []string               `json:"providers"`
	Categories         []string               `json:"categories"`
	SubCategories      []string               `json:"sub_categories"`
	Labels             map[string]string      `json:"labels"`
	CreatedAfter       *time.Time             `json:"created_after"`
	CreatedBefore      *time.Time             `json:"created_before"`
	UpdatedAfter       *time.Time             `json:"updated_after"`
	UpdatedBefore      *time.Time             `json:"updated_before"`
	Maturity           []models.MaturityLevel `json:"maturity"`
	VersionConstraint  string                 `json:"version_constraint"`
	MajorVersion       *int                   `json:"major_version"`
	DeploymentEngines  []string               `json:"deployment_engines"`
	HasDependency      string                 `json:"has_dependency"`
	ProvidesDependency string                 `json:"provides_dependency"`
}

type FacetResult struct {
	Values []FacetValue `json:"values"`
}

type FacetValue struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

type SortField string

const (
	SortByName     SortField = "name"
	SortByCreated  SortField = "created_at"
	SortByUpdated  SortField = "updated_at"
	SortByProvider SortField = "provider"
	SortByCategory SortField = "category"
	SortByVersion  SortField = "version"
	SortByMaturity SortField = "maturity"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)
