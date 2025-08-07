package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponent_GetID(t *testing.T) {
	component := &Component{
		Name:    "test-component",
		Version: "1.0.0",
	}

	expected := "test-component:1.0.0"
	assert.Equal(t, expected, component.GetID())
}

func TestComponent_IsDeprecated(t *testing.T) {
	tests := []struct {
		name      string
		component *Component
		expected  bool
	}{
		{
			name: "not deprecated",
			component: &Component{
				Metadata: ComponentMetadata{
					Deprecated: false,
				},
			},
			expected: false,
		},
		{
			name: "deprecated flag set",
			component: &Component{
				Metadata: ComponentMetadata{
					Deprecated: true,
				},
			},
			expected: true,
		},
		{
			name: "deprecated at set",
			component: &Component{
				Metadata: ComponentMetadata{
					Deprecated:   false,
					DeprecatedAt: &time.Time{},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.component.IsDeprecated())
		})
	}
}

func TestComponentValidator_Validate(t *testing.T) {
	validator := NewComponentValidator()

	tests := []struct {
		name      string
		component *Component
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "nil component",
			component: nil,
			wantErr:   true,
			errMsg:    "component cannot be nil",
		},
		{
			name: "valid component",
			component: &Component{
				Name:        "test-component",
				Version:     "1.0.0",
				Provider:    "aws",
				Category:    "database",
				Description: "Test component",
				Inputs: []InputSpec{
					{
						Name:        "db_name",
						Type:        "string",
						Description: "Database name",
					},
				},
				Outputs: []OutputSpec{
					{
						Name:        "endpoint",
						Type:        "string",
						Description: "Database endpoint",
					},
				},
				Deployment: DeploymentSpec{
					Engine:  "terraform",
					Version: "1.0.0",
				},
			},
			wantErr: false,
		},
		{
			name: "missing required name",
			component: &Component{
				Version:  "1.0.0",
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{
						Name:        "db_name",
						Type:        "string",
						Description: "Database name",
					},
				},
				Outputs: []OutputSpec{
					{
						Name:        "endpoint",
						Type:        "string",
						Description: "Database endpoint",
					},
				},
				Deployment: DeploymentSpec{
					Engine:  "terraform",
					Version: "1.0.0",
				},
			},
			wantErr: true,
			errMsg:  "validation failed",
		},
		{
			name: "invalid semantic version",
			component: &Component{
				Name:     "test-component",
				Version:  "invalid-version",
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{
						Name:        "db_name",
						Type:        "string",
						Description: "Database name",
					},
				},
				Outputs: []OutputSpec{
					{
						Name:        "endpoint",
						Type:        "string",
						Description: "Database endpoint",
					},
				},
				Deployment: DeploymentSpec{
					Engine:  "terraform",
					Version: "1.0.0",
				},
			},
			wantErr: true,
			errMsg:  "validation failed",
		},
		{
			name: "invalid DNS1123 name",
			component: &Component{
				Name:     "Test_Component",
				Version:  "1.0.0",
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{
						Name:        "db_name",
						Type:        "string",
						Description: "Database name",
					},
				},
				Outputs: []OutputSpec{
					{
						Name:        "endpoint",
						Type:        "string",
						Description: "Database endpoint",
					},
				},
				Deployment: DeploymentSpec{
					Engine:  "terraform",
					Version: "1.0.0",
				},
			},
			wantErr: true,
			errMsg:  "validation failed",
		},
		{
			name: "missing inputs",
			component: &Component{
				Name:     "test-component",
				Version:  "1.0.0",
				Provider: "aws",
				Category: "database",
				Inputs:   []InputSpec{},
				Outputs: []OutputSpec{
					{
						Name:        "endpoint",
						Type:        "string",
						Description: "Database endpoint",
					},
				},
				Deployment: DeploymentSpec{
					Engine:  "terraform",
					Version: "1.0.0",
				},
			},
			wantErr: true,
			errMsg:  "validation failed",
		},
		{
			name: "missing outputs",
			component: &Component{
				Name:     "test-component",
				Version:  "1.0.0",
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{
						Name:        "db_name",
						Type:        "string",
						Description: "Database name",
					},
				},
				Outputs: []OutputSpec{},
				Deployment: DeploymentSpec{
					Engine:  "terraform",
					Version: "1.0.0",
				},
			},
			wantErr: true,
			errMsg:  "validation failed",
		},
		{
			name: "input missing name",
			component: &Component{
				Name:     "test-component",
				Version:  "1.0.0",
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{
						Type:        "string",
						Description: "Database name",
					},
				},
				Outputs: []OutputSpec{
					{
						Name:        "endpoint",
						Type:        "string",
						Description: "Database endpoint",
					},
				},
				Deployment: DeploymentSpec{
					Engine:  "terraform",
					Version: "1.0.0",
				},
			},
			wantErr: true,
			errMsg:  "input at index 0 is missing name",
		},
		{
			name: "input missing type",
			component: &Component{
				Name:     "test-component",
				Version:  "1.0.0",
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{
						Name:        "db_name",
						Description: "Database name",
					},
				},
				Outputs: []OutputSpec{
					{
						Name:        "endpoint",
						Type:        "string",
						Description: "Database endpoint",
					},
				},
				Deployment: DeploymentSpec{
					Engine:  "terraform",
					Version: "1.0.0",
				},
			},
			wantErr: true,
			errMsg:  "input 'db_name' is missing type",
		},
		{
			name: "input missing description",
			component: &Component{
				Name:     "test-component",
				Version:  "1.0.0",
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{
						Name: "db_name",
						Type: "string",
					},
				},
				Outputs: []OutputSpec{
					{
						Name:        "endpoint",
						Type:        "string",
						Description: "Database endpoint",
					},
				},
				Deployment: DeploymentSpec{
					Engine:  "terraform",
					Version: "1.0.0",
				},
			},
			wantErr: true,
			errMsg:  "input 'db_name' is missing description",
		},
		{
			name: "output missing name",
			component: &Component{
				Name:     "test-component",
				Version:  "1.0.0",
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{
						Name:        "db_name",
						Type:        "string",
						Description: "Database name",
					},
				},
				Outputs: []OutputSpec{
					{
						Type:        "string",
						Description: "Database endpoint",
					},
				},
				Deployment: DeploymentSpec{
					Engine:  "terraform",
					Version: "1.0.0",
				},
			},
			wantErr: true,
			errMsg:  "output at index 0 is missing name",
		},
		{
			name: "output missing type",
			component: &Component{
				Name:     "test-component",
				Version:  "1.0.0",
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{
						Name:        "db_name",
						Type:        "string",
						Description: "Database name",
					},
				},
				Outputs: []OutputSpec{
					{
						Name:        "endpoint",
						Description: "Database endpoint",
					},
				},
				Deployment: DeploymentSpec{
					Engine:  "terraform",
					Version: "1.0.0",
				},
			},
			wantErr: true,
			errMsg:  "output 'endpoint' is missing type",
		},
		{
			name: "output missing description",
			component: &Component{
				Name:     "test-component",
				Version:  "1.0.0",
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{
						Name:        "db_name",
						Type:        "string",
						Description: "Database name",
					},
				},
				Outputs: []OutputSpec{
					{
						Name: "endpoint",
						Type: "string",
					},
				},
				Deployment: DeploymentSpec{
					Engine:  "terraform",
					Version: "1.0.0",
				},
			},
			wantErr: true,
			errMsg:  "output 'endpoint' is missing description",
		},
		{
			name: "deployment missing engine",
			component: &Component{
				Name:     "test-component",
				Version:  "1.0.0",
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{
						Name:        "db_name",
						Type:        "string",
						Description: "Database name",
					},
				},
				Outputs: []OutputSpec{
					{
						Name:        "endpoint",
						Type:        "string",
						Description: "Database endpoint",
					},
				},
				Deployment: DeploymentSpec{
					Version: "1.0.0",
				},
			},
			wantErr: true,
			errMsg:  "validation failed",
		},
		{
			name: "deployment missing version",
			component: &Component{
				Name:     "test-component",
				Version:  "1.0.0",
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{
						Name:        "db_name",
						Type:        "string",
						Description: "Database name",
					},
				},
				Outputs: []OutputSpec{
					{
						Name:        "endpoint",
						Type:        "string",
						Description: "Database endpoint",
					},
				},
				Deployment: DeploymentSpec{
					Engine: "terraform",
				},
			},
			wantErr: true,
			errMsg:  "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.component)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateSemanticVersion(t *testing.T) {
	validator := NewComponentValidator()

	tests := []struct {
		name    string
		version string
		valid   bool
	}{
		{"valid version", "1.0.0", true},
		{"valid version with v prefix", "v1.0.0", true},
		{"valid pre-release", "1.0.0-alpha", true},
		{"valid build metadata", "1.0.0+build.1", true},
		{"valid complex version", "1.0.0-alpha.1+build.1", true},
		{"invalid format", "1.0", false},
		{"invalid format", "1", false},
		{"invalid format", "1.0.0.0", false},
		{"invalid characters", "1.0.a", false},
		{"empty version", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := &Component{
				Name:     "test-component",
				Version:  tt.version,
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{
						Name:        "db_name",
						Type:        "string",
						Description: "Database name",
					},
				},
				Outputs: []OutputSpec{
					{
						Name:        "endpoint",
						Type:        "string",
						Description: "Database endpoint",
					},
				},
				Deployment: DeploymentSpec{
					Engine:  "terraform",
					Version: "1.0.0",
				},
			}

			err := validator.Validate(component)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "validation failed")
			}
		})
	}
}

func TestValidateDNS1123(t *testing.T) {
	validator := NewComponentValidator()

	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"valid name", "test-component", true},
		{"valid name with numbers", "test-component-123", true},
		{"valid single char", "a", true},
		{"valid with hyphens", "test-comp-name", true},
		{"invalid uppercase", "Test-Component", false},
		{"invalid underscore", "test_component", false},
		{"invalid start with hyphen", "-test-component", false},
		{"invalid end with hyphen", "test-component-", false},
		{"invalid empty", "", false},
		{"invalid too long", "this-is-a-very-long-component-name-that-exceeds-the-maximum-length-allowed-for-dns1123-names", false},
		{"invalid special chars", "test@component", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component := &Component{
				Name:     tt.value,
				Version:  "1.0.0",
				Provider: "aws",
				Category: "database",
				Inputs: []InputSpec{
					{
						Name:        "db_name",
						Type:        "string",
						Description: "Database name",
					},
				},
				Outputs: []OutputSpec{
					{
						Name:        "endpoint",
						Type:        "string",
						Description: "Database endpoint",
					},
				},
				Deployment: DeploymentSpec{
					Engine:  "terraform",
					Version: "1.0.0",
				},
			}

			err := validator.Validate(component)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "validation failed")
			}
		})
	}
}
