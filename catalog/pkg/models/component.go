package models

import (
	"fmt"
	"slices"
	"time"

	"github.com/HatiCode/nestor/shared/pkg/json"
)

type ComponentDefinition struct {
	Metadata ComponentMetadata `json:"metadata"`
	Spec     ComponentSpec     `json:"spec"`
	Status   ComponentStatus   `json:"status"`
}

type ComponentMetadata struct {
	Name              string            `json:"name" validate:"required,dns1123"`
	DisplayName       string            `json:"display_name" validate:"required"`
	Description       string            `json:"description" validate:"required"`
	Version           string            `json:"version" validate:"required,semver"`
	Provider          string            `json:"provider" validate:"required"`
	Category          string            `json:"category" validate:"required"`
	SubCategory       string            `json:"sub_category"`
	ResourceType      string            `json:"resource_type" validate:"required"`
	DeploymentEngines []string          `json:"deployment_engines" validate:"required,min=1"`
	Maturity          MaturityLevel     `json:"maturity" validate:"required"` // alpha, beta, stable, deprecated
	Maintainers       []string          `json:"maintainers" validate:"required,min=1"`
	Documentation     []DocLink         `json:"documentation"`
	CostEstimate      CostEstimate      `json:"cost_estimate"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	DeprecatedAt      *time.Time        `json:"deprecated_at,omitempty"`
	GitRepository     string            `json:"git_repository"`
	GitPath           string            `json:"git_path"`
	GitCommit         string            `json:"git_commit"`
	GitBranch         string            `json:"git_branch"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
}

type ComponentSpec struct {
	Dependencies   []Dependency          `json:"dependencies"`
	Provides       []string              `json:"provides"`
	ConflictsWith  []string              `json:"conflicts_with"`
	RequiredInputs []InputSpec           `json:"required_inputs"`
	OptionalInputs []InputSpec           `json:"optional_inputs"`
	Outputs        []OutputSpec          `json:"outputs"`
	EngineSpecs    map[string]EngineSpec `json:"engine_specs"`
}

type ComponentStatus struct {
	State            ComponentState   `json:"state"`
	UsageCount       int64            `json:"usage_count"`
	LastUsed         *time.Time       `json:"last_used"`
	ValidationStatus ValidationStatus `json:"validation_status"`
	HealthStatus     HealthStatus     `json:"health_status"`
	Stats            ComponentStats   `json:"stats"`
}

type Dependency struct {
	Name        string `json:"name" validate:"required"`
	Type        string `json:"type" validate:"required"`
	Version     string `json:"version" validate:"required"`
	Optional    bool   `json:"optional"`
	Description string `json:"description"`
	Condition   string `json:"condition,omitempty"`
}

type InputSpec struct {
	Name        string     `json:"name" validate:"required"`
	Type        string     `json:"type" validate:"required"`
	Description string     `json:"description" validate:"required"`
	Default     any        `json:"default,omitempty"`
	Validation  Validation `json:"validation"`
	Sensitive   bool       `json:"sensitive"`
	Examples    []string   `json:"examples"`
	Group       string     `json:"group"`
}

type OutputSpec struct {
	Name        string `json:"name" validate:"required"`
	Type        string `json:"type" validate:"required"`
	Description string `json:"description" validate:"required"`
	Sensitive   bool   `json:"sensitive"`
	Export      bool   `json:"export"`
}

type EngineSpec struct {
	Engine       string         `json:"engine"`
	Version      string         `json:"version"`
	Template     string         `json:"template"`
	Config       map[string]any `json:"config"`
	Dependencies []string       `json:"dependencies"`
}

type Validation struct {
	Required  bool     `json:"required"`
	MinLength *int     `json:"min_length,omitempty"`
	MaxLength *int     `json:"max_length,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Enum      []string `json:"enum,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Custom    string   `json:"custom,omitempty"`
}

type ValidationRule struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Expression   string `json:"expression"`
	ErrorMessage string `json:"error_message"`
}

type MaturityLevel string

const (
	MaturityAlpha      MaturityLevel = "alpha"
	MaturityBeta       MaturityLevel = "beta"
	MaturityStable     MaturityLevel = "stable"
	MaturityDeprecated MaturityLevel = "deprecated"
)

type ComponentState string

const (
	ComponentStateActive     ComponentState = "active"
	ComponentStateDeprecated ComponentState = "deprecated"
	ComponentStateArchived   ComponentState = "archived"
)

type ValidationStatus string

const (
	ValidationStatusValid   ValidationStatus = "valid"
	ValidationStatusInvalid ValidationStatus = "invalid"
	ValidationStatusPending ValidationStatus = "pending"
	ValidationStatusUnknown ValidationStatus = "unknown"
)

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

type DocLink struct {
	Title string `json:"title" validate:"required"`
	URL   string `json:"url" validate:"required"`
	Type  string `json:"type"`
}

type CostEstimate struct {
	Currency    string    `json:"currency"`
	HourlyCost  float64   `json:"hourly_cost"`
	MonthlyCost float64   `json:"monthly_cost"`
	CostModel   string    `json:"cost_model"`
	LastUpdated time.Time `json:"last_updated"`
	Region      string    `json:"region"`
	Notes       string    `json:"notes"`
}

type ComponentStats struct {
	TotalDeployments      int64      `json:"total_deployments"`
	ActiveDeployments     int64      `json:"active_deployments"`
	SuccessfulDeployments int64      `json:"successful_deployments"`
	FailedDeployments     int64      `json:"failed_deployments"`
	AverageDeployTime     *float64   `json:"average_deploy_time_seconds"`
	LastDeploymentAt      *time.Time `json:"last_deployment_at"`
	PopularityScore       float64    `json:"popularity_score"`
}

func (c *ComponentDefinition) GetID() string {
	return c.Metadata.Name + ":" + c.Metadata.Version
}

func (c *ComponentDefinition) IsDeprecated() bool {
	return c.Metadata.DeprecatedAt != nil || c.Metadata.Maturity == MaturityDeprecated
}

func (c *ComponentDefinition) IsStable() bool {
	return c.Metadata.Maturity == MaturityStable
}

func (c *ComponentDefinition) SupportsEngine(engine string) bool {
	return slices.Contains(c.Metadata.DeploymentEngines, engine)
}

func (c *ComponentDefinition) GetEngineSpec(engine string) (*EngineSpec, bool) {
	spec, exists := c.Spec.EngineSpecs[engine]
	return &spec, exists
}

func (c *ComponentDefinition) HasLabel(key, value string) bool {
	if c.Metadata.Labels == nil {
		return false
	}
	labelValue, exists := c.Metadata.Labels[key]
	return exists && labelValue == value
}

func (c *ComponentDefinition) GetDependenciesOfType(depType string) []Dependency {
	var result []Dependency
	for _, dep := range c.Spec.Dependencies {
		if dep.Type == depType {
			result = append(result, dep)
		}
	}
	return result
}

func (c *ComponentDefinition) MarshalJSON() ([]byte, error) {
	type Alias ComponentDefinition
	return json.ToJSON(&struct {
		*Alias
		ID string `json:"id"`
	}{
		Alias: (*Alias)(c),
		ID:    c.GetID(),
	})
}

func (c *ComponentDefinition) Validate() error {
	if c.Metadata.Name == "" {
		return ErrInvalidComponentName
	}
	if c.Metadata.Version == "" {
		return ErrInvalidComponentVersion
	}
	if len(c.Metadata.DeploymentEngines) == 0 {
		return ErrNoDeploymentEngines
	}
	return nil
}

var (
	ErrInvalidComponentName    = fmt.Errorf("component name is required")
	ErrInvalidComponentVersion = fmt.Errorf("component version is required")
	ErrNoDeploymentEngines     = fmt.Errorf("at least one deployment engine is required")
)
