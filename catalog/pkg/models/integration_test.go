package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_ComponentValidationWorkflow tests the complete workflow
func TestIntegration_ComponentValidationWorkflow(t *testing.T) {
	// Create a validator
	validator := NewComponentValidator()

	// Create a sample component
	component := &Component{
		Name:        "postgres-database",
		Version:     "1.2.3",
		Provider:    "aws",
		Category:    "database",
		Description: "PostgreSQL database component for AWS RDS",
		Inputs: []InputSpec{
			{
				Name:        "db_name",
				Type:        "string",
				Description: "Name of the database",
				Validation: Validation{
					Required:  true,
					MinLength: &[]int{3}[0],
					MaxLength: &[]int{63}[0],
				},
			},
			{
				Name:        "instance_class",
				Type:        "string",
				Description: "RDS instance class",
				Default:     "db.t3.micro",
				Validation: Validation{
					Required: false,
					Enum:     []string{"db.t3.micro", "db.t3.small", "db.t3.medium"},
				},
			},
		},
		Outputs: []OutputSpec{
			{
				Name:        "endpoint",
				Type:        "string",
				Description: "Database connection endpoint",
				Sensitive:   false,
			},
			{
				Name:        "password",
				Type:        "string",
				Description: "Database master password",
				Sensitive:   true,
			},
		},
		Deployment: DeploymentSpec{
			Engine:  "terraform",
			Version: "1.5.0",
			Config: map[string]any{
				"provider": "aws",
				"region":   "us-east-1",
			},
		},
		Metadata: ComponentMetadata{
			GitCommit:  "abc123def456",
			Deprecated: false,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Test validation
	err := validator.Validate(component)
	require.NoError(t, err, "Component should be valid")

	// Test component methods
	assert.Equal(t, "postgres-database:1.2.3", component.GetID())
	assert.False(t, component.IsDeprecated())

	// Test semantic version parsing
	semVer, err := ParseSemanticVersion(component.Version)
	require.NoError(t, err)
	assert.Equal(t, 1, semVer.Major)
	assert.Equal(t, 2, semVer.Minor)
	assert.Equal(t, 3, semVer.Patch)

	// Test version history
	versionHistory := &VersionHistory{
		ComponentName: component.Name,
		Versions: []Version{
			{
				Version:   "1.2.3",
				CreatedAt: time.Now(),
			},
			{
				Version:   "1.2.2",
				CreatedAt: time.Now().Add(-time.Hour),
			},
			{
				Version:      "1.2.1",
				CreatedAt:    time.Now().Add(-2 * time.Hour),
				Deprecated:   true,
				DeprecatedAt: &time.Time{},
			},
		},
		TotalCount: 3,
	}

	latest := versionHistory.GetLatestVersion()
	require.NotNil(t, latest)
	assert.Equal(t, "1.2.3", latest.Version)

	latestNonDeprecated := versionHistory.GetLatestNonDeprecatedVersion()
	require.NotNil(t, latestNonDeprecated)
	assert.Equal(t, "1.2.3", latestNonDeprecated.Version)
}

// TestIntegration_ValidationErrors tests various validation error scenarios
func TestIntegration_ValidationErrors(t *testing.T) {
	validator := NewComponentValidator()

	testCases := []struct {
		name      string
		component *Component
		expectErr string
	}{
		{
			name: "invalid DNS name",
			component: &Component{
				Name:     "Invalid_Name",
				Version:  "1.0.0",
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{Name: "test", Type: "string", Description: "test"},
				},
				Outputs: []OutputSpec{
					{Name: "test", Type: "string", Description: "test"},
				},
				Deployment: DeploymentSpec{Engine: "terraform", Version: "1.0.0"},
			},
			expectErr: "validation failed",
		},
		{
			name: "invalid semantic version",
			component: &Component{
				Name:     "valid-name",
				Version:  "not-a-version",
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{Name: "test", Type: "string", Description: "test"},
				},
				Outputs: []OutputSpec{
					{Name: "test", Type: "string", Description: "test"},
				},
				Deployment: DeploymentSpec{Engine: "terraform", Version: "1.0.0"},
			},
			expectErr: "validation failed",
		},
		{
			name: "missing required fields",
			component: &Component{
				Name:    "valid-name",
				Version: "1.0.0",
				// Missing Provider and Category
				Inputs: []InputSpec{
					{Name: "test", Type: "string", Description: "test"},
				},
				Outputs: []OutputSpec{
					{Name: "test", Type: "string", Description: "test"},
				},
				Deployment: DeploymentSpec{Engine: "terraform", Version: "1.0.0"},
			},
			expectErr: "validation failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.Validate(tc.component)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectErr)
		})
	}
}
