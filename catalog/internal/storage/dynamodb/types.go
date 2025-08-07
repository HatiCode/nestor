package dynamodb

import (
	"fmt"
	"time"

	"github.com/HatiCode/nestor/catalog/pkg/models"
)

// ComponentItem represents a component stored in DynamoDB.
type ComponentItem struct {
	// DynamoDB keys
	PK string `dynamodbav:"PK"`
	SK string `dynamodbav:"SK"`

	// Component metadata
	Name              string            `dynamodbav:"Name"`
	DisplayName       string            `dynamodbav:"DisplayName"`
	Description       string            `dynamodbav:"Description"`
	Version           string            `dynamodbav:"Version"`
	Provider          string            `dynamodbav:"Provider"`
	Category          string            `dynamodbav:"Category"`
	SubCategory       string            `dynamodbav:"SubCategory"`
	ResourceType      string            `dynamodbav:"ResourceType"`
	DeploymentEngines []string          `dynamodbav:"DeploymentEngines"`
	Maturity          string            `dynamodbav:"Maturity"`
	Maintainers       []string          `dynamodbav:"Maintainers"`
	Documentation     []models.DocLink  `dynamodbav:"Documentation"`
	CreatedAt         time.Time         `dynamodbav:"CreatedAt"`
	UpdatedAt         time.Time         `dynamodbav:"UpdatedAt"`
	DeprecatedAt      *time.Time        `dynamodbav:"DeprecatedAt,omitempty"`
	GitRepository     string            `dynamodbav:"GitRepository"`
	GitPath           string            `dynamodbav:"GitPath"`
	GitCommit         string            `dynamodbav:"GitCommit"`
	GitBranch         string            `dynamodbav:"GitBranch"`
	Labels            map[string]string `dynamodbav:"Labels"`
	Annotations       map[string]string `dynamodbav:"Annotations"`

	// Component spec
	Dependencies   []models.Dependency          `dynamodbav:"Dependencies"`
	Provides       []string                     `dynamodbav:"Provides"`
	ConflictsWith  []string                     `dynamodbav:"ConflictsWith"`
	RequiredInputs []models.InputSpec           `dynamodbav:"RequiredInputs"`
	OptionalInputs []models.InputSpec           `dynamodbav:"OptionalInputs"`
	Outputs        []models.OutputSpec          `dynamodbav:"Outputs"`
	EngineSpecs    map[string]models.EngineSpec `dynamodbav:"EngineSpecs"`

	// Component status
	State            string                `dynamodbav:"State"`
	UsageCount       int64                 `dynamodbav:"UsageCount"`
	LastUsed         *time.Time            `dynamodbav:"LastUsed,omitempty"`
	ValidationStatus string                `dynamodbav:"ValidationStatus"`
	HealthStatus     string                `dynamodbav:"HealthStatus"`
	Stats            models.ComponentStats `dynamodbav:"Stats"`

	// Additional fields for querying
	GSI1PK string `dynamodbav:"GSI1PK"`
	GSI1SK string `dynamodbav:"GSI1SK"`
}

// ToComponent converts a DynamoDB item to a Component.
func (item *ComponentItem) ToComponent() *models.Component {
	// For MVP, we'll map the complex DynamoDB structure to the simplified Component model
	// Combine RequiredInputs and OptionalInputs into a single Inputs slice
	inputs := make([]models.InputSpec, 0, len(item.RequiredInputs)+len(item.OptionalInputs))
	inputs = append(inputs, item.RequiredInputs...)
	inputs = append(inputs, item.OptionalInputs...)

	// Create a simplified deployment spec from the first engine spec
	deployment := models.DeploymentSpec{
		Engine:  "terraform", // Default for MVP
		Version: "1.0.0",     // Default for MVP
		Config:  make(map[string]any),
	}

	// If we have engine specs, use the first one
	if len(item.EngineSpecs) > 0 {
		for engine, spec := range item.EngineSpecs {
			deployment.Engine = engine
			deployment.Version = spec.Version
			deployment.Config = spec.Config
			break // Use first engine spec for MVP
		}
	}

	return &models.Component{
		Name:        item.Name,
		Version:     item.Version,
		Provider:    item.Provider,
		Category:    item.Category,
		Description: item.Description,
		Inputs:      inputs,
		Outputs:     item.Outputs,
		Deployment:  deployment,
		Metadata: models.ComponentMetadata{
			GitCommit:    item.GitCommit,
			Deprecated:   item.DeprecatedAt != nil,
			DeprecatedAt: item.DeprecatedAt,
		},
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

// NewComponentItemFromComponent creates a new ComponentItem from a Component.
func NewComponentItemFromComponent(component *models.Component) *ComponentItem {
	if component == nil {
		return nil
	}

	now := time.Now()
	if component.CreatedAt.IsZero() {
		component.CreatedAt = now
	}
	component.UpdatedAt = now

	// For MVP, split inputs into required and optional based on validation rules
	var requiredInputs, optionalInputs []models.InputSpec
	for _, input := range component.Inputs {
		if input.Validation.Required {
			requiredInputs = append(requiredInputs, input)
		} else {
			optionalInputs = append(optionalInputs, input)
		}
	}

	// Create engine specs from deployment spec
	engineSpecs := make(map[string]models.EngineSpec)
	engineSpecs[component.Deployment.Engine] = models.EngineSpec{
		Engine:  component.Deployment.Engine,
		Version: component.Deployment.Version,
		Config:  component.Deployment.Config,
	}

	item := &ComponentItem{
		// Component metadata - map from simplified model
		Name:              component.Name,
		DisplayName:       component.Name, // Use name as display name for MVP
		Description:       component.Description,
		Version:           component.Version,
		Provider:          component.Provider,
		Category:          component.Category,
		SubCategory:       "",                                    // Not in MVP model
		ResourceType:      "infrastructure",                      // Default for MVP
		DeploymentEngines: []string{component.Deployment.Engine}, // Single engine for MVP
		Maturity:          "stable",                              // Default for MVP
		Maintainers:       []string{},                            // Empty for MVP
		Documentation:     []models.DocLink{},                    // Empty for MVP
		CreatedAt:         component.CreatedAt,
		UpdatedAt:         component.UpdatedAt,
		DeprecatedAt:      component.Metadata.DeprecatedAt,
		GitRepository:     "", // Not in MVP model
		GitPath:           "", // Not in MVP model
		GitCommit:         component.Metadata.GitCommit,
		GitBranch:         "",                      // Not in MVP model
		Labels:            make(map[string]string), // Empty for MVP
		Annotations:       make(map[string]string), // Empty for MVP

		// Component spec - map from simplified model
		Dependencies:   []models.Dependency{}, // Empty for MVP
		Provides:       []string{},            // Empty for MVP
		ConflictsWith:  []string{},            // Empty for MVP
		RequiredInputs: requiredInputs,
		OptionalInputs: optionalInputs,
		Outputs:        component.Outputs,
		EngineSpecs:    engineSpecs,

		// Component status - defaults for MVP
		State:            "active",                // Default for MVP
		UsageCount:       0,                       // Default for MVP
		LastUsed:         nil,                     // Default for MVP
		ValidationStatus: "valid",                 // Default for MVP
		HealthStatus:     "healthy",               // Default for MVP
		Stats:            models.ComponentStats{}, // Empty for MVP
	}

	// Set the DynamoDB keys
	item.PK = fmt.Sprintf("COMPONENT#%s", component.Name)
	item.SK = fmt.Sprintf("VERSION#%s", component.Version)

	// Set GSI keys for querying
	item.GSI1PK = fmt.Sprintf("PROVIDER#%s", component.Provider)
	item.GSI1SK = fmt.Sprintf("CATEGORY#%s", component.Category)

	return item
}
