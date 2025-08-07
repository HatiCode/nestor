package models

import (
	"fmt"
	"time"

	"github.com/HatiCode/nestor/shared/pkg/json"
	"github.com/go-playground/validator/v10"
)

// Component represents a simplified infrastructure component definition for the MVP
type Component struct {
	Name        string            `json:"name" validate:"required,dns1123"`
	Version     string            `json:"version" validate:"required,semver"`
	Provider    string            `json:"provider" validate:"required"`
	Category    string            `json:"category" validate:"required"`
	Description string            `json:"description"`
	Inputs      []InputSpec       `json:"inputs" validate:"required,min=1"`
	Outputs     []OutputSpec      `json:"outputs" validate:"required,min=1"`
	Deployment  DeploymentSpec    `json:"deployment" validate:"required"`
	Metadata    ComponentMetadata `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// ComponentMetadata contains additional metadata for the component
type ComponentMetadata struct {
	GitCommit    string     `json:"git_commit,omitempty"`
	Deprecated   bool       `json:"deprecated"`
	DeprecatedAt *time.Time `json:"deprecated_at,omitempty"`
}

// InputSpec defines an input parameter specification
type InputSpec struct {
	Name        string     `json:"name" validate:"required"`
	Type        string     `json:"type" validate:"required"`
	Description string     `json:"description" validate:"required"`
	Default     any        `json:"default,omitempty"`
	Validation  Validation `json:"validation"`
	Sensitive   bool       `json:"sensitive"`
}

// OutputSpec defines an output parameter specification
type OutputSpec struct {
	Name        string `json:"name" validate:"required"`
	Type        string `json:"type" validate:"required"`
	Description string `json:"description" validate:"required"`
	Sensitive   bool   `json:"sensitive"`
}

// DeploymentSpec defines deployment engine specifications
type DeploymentSpec struct {
	Engine  string         `json:"engine" validate:"required"`
	Version string         `json:"version" validate:"required"`
	Config  map[string]any `json:"config"`
}

// Validation defines validation rules for input parameters
type Validation struct {
	Required  bool     `json:"required"`
	MinLength *int     `json:"min_length,omitempty"`
	MaxLength *int     `json:"max_length,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Enum      []string `json:"enum,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
}

// Legacy types for backward compatibility with existing storage layer
type EngineSpec struct {
	Engine       string         `json:"engine"`
	Version      string         `json:"version"`
	Template     string         `json:"template"`
	Config       map[string]any `json:"config"`
	Dependencies []string       `json:"dependencies"`
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

type DocLink struct {
	Title string `json:"title" validate:"required"`
	URL   string `json:"url" validate:"required"`
	Type  string `json:"type"`
}

type Dependency struct {
	Name        string `json:"name" validate:"required"`
	Type        string `json:"type" validate:"required"`
	Version     string `json:"version" validate:"required"`
	Optional    bool   `json:"optional"`
	Description string `json:"description"`
	Condition   string `json:"condition,omitempty"`
}

// ComponentValidator provides validation functionality for components
type ComponentValidator struct {
	validator *validator.Validate
}

// NewComponentValidator creates a new component validator with custom validation rules
func NewComponentValidator() *ComponentValidator {
	v := validator.New()

	// Register custom validation functions
	v.RegisterValidation("semver", validateSemanticVersion)
	v.RegisterValidation("dns1123", validateDNS1123)

	return &ComponentValidator{
		validator: v,
	}
}

// GetID returns a unique identifier for the component
func (c *Component) GetID() string {
	return c.Name + ":" + c.Version
}

// IsDeprecated returns true if the component is deprecated
func (c *Component) IsDeprecated() bool {
	return c.Metadata.Deprecated || c.Metadata.DeprecatedAt != nil
}

// MarshalJSON adds the ID field to the JSON output
func (c *Component) MarshalJSON() ([]byte, error) {
	type Alias Component
	return json.ToJSON(&struct {
		*Alias
		ID string `json:"id"`
	}{
		Alias: (*Alias)(c),
		ID:    c.GetID(),
	})
}

// Validate validates a component according to the business rules
func (cv *ComponentValidator) Validate(component *Component) error {
	if component == nil {
		return fmt.Errorf("component cannot be nil")
	}

	// Perform struct validation
	if err := cv.validator.Struct(component); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Additional business logic validation
	if err := cv.validateInputsAndOutputs(component); err != nil {
		return err
	}

	if err := cv.validateDeploymentSpec(component); err != nil {
		return err
	}

	return nil
}

// validateInputsAndOutputs validates input and output specifications
func (cv *ComponentValidator) validateInputsAndOutputs(component *Component) error {
	// Validate inputs
	for i, input := range component.Inputs {
		if input.Name == "" {
			return fmt.Errorf("input at index %d is missing name", i)
		}
		if input.Type == "" {
			return fmt.Errorf("input '%s' is missing type", input.Name)
		}
		if input.Description == "" {
			return fmt.Errorf("input '%s' is missing description", input.Name)
		}
	}

	// Validate outputs
	for i, output := range component.Outputs {
		if output.Name == "" {
			return fmt.Errorf("output at index %d is missing name", i)
		}
		if output.Type == "" {
			return fmt.Errorf("output '%s' is missing type", output.Name)
		}
		if output.Description == "" {
			return fmt.Errorf("output '%s' is missing description", output.Name)
		}
	}

	return nil
}

// validateDeploymentSpec validates deployment engine specifications
func (cv *ComponentValidator) validateDeploymentSpec(component *Component) error {
	if component.Deployment.Engine == "" {
		return fmt.Errorf("deployment engine is required")
	}
	if component.Deployment.Version == "" {
		return fmt.Errorf("deployment engine version is required")
	}
	return nil
}

// validateSemanticVersion validates semantic version format
func validateSemanticVersion(fl validator.FieldLevel) bool {
	version := fl.Field().String()
	_, err := ParseSemanticVersion(version)
	return err == nil
}

// validateDNS1123 validates DNS-1123 compliant names
func validateDNS1123(fl validator.FieldLevel) bool {
	name := fl.Field().String()
	if len(name) == 0 || len(name) > 63 {
		return false
	}

	// DNS-1123 names must start and end with alphanumeric characters
	// and can contain hyphens in the middle
	for i, r := range name {
		if i == 0 || i == len(name)-1 {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
				return false
			}
		} else {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
				return false
			}
		}
	}

	return true
}

var (
	ErrInvalidComponentName    = fmt.Errorf("component name is required")
	ErrInvalidComponentVersion = fmt.Errorf("component version is required")
	ErrNoDeploymentEngines     = fmt.Errorf("at least one deployment engine is required")
)
