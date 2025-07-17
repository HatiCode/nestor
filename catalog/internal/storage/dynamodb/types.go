package dynamodb

import (
	"fmt"
	"time"

	"github.com/HatiCode/nestor/catalog/pkg/models"
)

// ComponentItem represents a component stored in DynamoDB
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

// ToComponentDefinition converts a DynamoDB item to a ComponentDefinition
func (item *ComponentItem) ToComponentDefinition() *models.ComponentDefinition {
	return &models.ComponentDefinition{
		Metadata: models.ComponentMetadata{
			Name:              item.Name,
			DisplayName:       item.DisplayName,
			Description:       item.Description,
			Version:           item.Version,
			Provider:          item.Provider,
			Category:          item.Category,
			SubCategory:       item.SubCategory,
			ResourceType:      item.ResourceType,
			DeploymentEngines: item.DeploymentEngines,
			Maturity:          models.MaturityLevel(item.Maturity),
			Maintainers:       item.Maintainers,
			Documentation:     item.Documentation,
			CreatedAt:         item.CreatedAt,
			UpdatedAt:         item.UpdatedAt,
			DeprecatedAt:      item.DeprecatedAt,
			GitRepository:     item.GitRepository,
			GitPath:           item.GitPath,
			GitCommit:         item.GitCommit,
			GitBranch:         item.GitBranch,
			Labels:            item.Labels,
			Annotations:       item.Annotations,
		},
		Spec: models.ComponentSpec{
			Dependencies:   item.Dependencies,
			Provides:       item.Provides,
			ConflictsWith:  item.ConflictsWith,
			RequiredInputs: item.RequiredInputs,
			OptionalInputs: item.OptionalInputs,
			Outputs:        item.Outputs,
			EngineSpecs:    item.EngineSpecs,
		},
		Status: models.ComponentStatus{
			State:            models.ComponentState(item.State),
			UsageCount:       item.UsageCount,
			LastUsed:         item.LastUsed,
			ValidationStatus: models.ValidationStatus(item.ValidationStatus),
			HealthStatus:     models.HealthStatus(item.HealthStatus),
			Stats:            item.Stats,
		},
	}
}

// NewComponentItemFromDefinition creates a new ComponentItem from a ComponentDefinition
func NewComponentItemFromDefinition(component *models.ComponentDefinition) *ComponentItem {
	if component == nil {
		return nil
	}

	now := time.Now()
	if component.Metadata.CreatedAt.IsZero() {
		component.Metadata.CreatedAt = now
	}
	component.Metadata.UpdatedAt = now

	item := &ComponentItem{
		// Component metadata
		Name:              component.Metadata.Name,
		DisplayName:       component.Metadata.DisplayName,
		Description:       component.Metadata.Description,
		Version:           component.Metadata.Version,
		Provider:          component.Metadata.Provider,
		Category:          component.Metadata.Category,
		SubCategory:       component.Metadata.SubCategory,
		ResourceType:      component.Metadata.ResourceType,
		DeploymentEngines: component.Metadata.DeploymentEngines,
		Maturity:          string(component.Metadata.Maturity),
		Maintainers:       component.Metadata.Maintainers,
		Documentation:     component.Metadata.Documentation,
		CreatedAt:         component.Metadata.CreatedAt,
		UpdatedAt:         component.Metadata.UpdatedAt,
		DeprecatedAt:      component.Metadata.DeprecatedAt,
		GitRepository:     component.Metadata.GitRepository,
		GitPath:           component.Metadata.GitPath,
		GitCommit:         component.Metadata.GitCommit,
		GitBranch:         component.Metadata.GitBranch,
		Labels:            component.Metadata.Labels,
		Annotations:       component.Metadata.Annotations,

		// Component spec
		Dependencies:   component.Spec.Dependencies,
		Provides:       component.Spec.Provides,
		ConflictsWith:  component.Spec.ConflictsWith,
		RequiredInputs: component.Spec.RequiredInputs,
		OptionalInputs: component.Spec.OptionalInputs,
		Outputs:        component.Spec.Outputs,
		EngineSpecs:    component.Spec.EngineSpecs,

		// Component status
		State:            string(component.Status.State),
		UsageCount:       component.Status.UsageCount,
		LastUsed:         component.Status.LastUsed,
		ValidationStatus: string(component.Status.ValidationStatus),
		HealthStatus:     string(component.Status.HealthStatus),
		Stats:            component.Status.Stats,
	}

	// Set the DynamoDB keys
	item.PK = fmt.Sprintf("COMPONENT#%s", component.Metadata.Name)
	item.SK = fmt.Sprintf("VERSION#%s", component.Metadata.Version)

	// Set GSI keys for querying
	item.GSI1PK = fmt.Sprintf("PROVIDER#%s", component.Metadata.Provider)
	item.GSI1SK = fmt.Sprintf("CATEGORY#%s", component.Metadata.Category)

	return item
}
