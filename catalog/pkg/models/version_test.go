package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion_IsDeprecated(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		version  *Version
		expected bool
	}{
		{
			name: "not deprecated",
			version: &Version{
				Version:    "1.0.0",
				Deprecated: false,
			},
			expected: false,
		},
		{
			name: "deprecated flag set",
			version: &Version{
				Version:    "1.0.0",
				Deprecated: true,
			},
			expected: true,
		},
		{
			name: "deprecated at set",
			version: &Version{
				Version:      "1.0.0",
				Deprecated:   false,
				DeprecatedAt: &now,
			},
			expected: true,
		},
		{
			name: "both deprecated flag and deprecated at set",
			version: &Version{
				Version:      "1.0.0",
				Deprecated:   true,
				DeprecatedAt: &now,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.version.IsDeprecated())
		})
	}
}

func TestVersion_GetSemanticVersion(t *testing.T) {
	tests := []struct {
		name        string
		version     *Version
		expectError bool
		expected    *SemanticVersionInfo
	}{
		{
			name: "valid semantic version",
			version: &Version{
				Version: "1.2.3",
			},
			expectError: false,
			expected: &SemanticVersionInfo{
				Major: 1,
				Minor: 2,
				Patch: 3,
				Raw:   "1.2.3",
			},
		},
		{
			name: "valid semantic version with pre-release",
			version: &Version{
				Version: "1.2.3-alpha",
			},
			expectError: false,
			expected: &SemanticVersionInfo{
				Major:      1,
				Minor:      2,
				Patch:      3,
				PreRelease: "alpha",
				Raw:        "1.2.3-alpha",
			},
		},
		{
			name: "invalid semantic version",
			version: &Version{
				Version: "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.version.GetSemanticVersion()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected.Major, result.Major)
				assert.Equal(t, tt.expected.Minor, result.Minor)
				assert.Equal(t, tt.expected.Patch, result.Patch)
				assert.Equal(t, tt.expected.PreRelease, result.PreRelease)
				assert.Equal(t, tt.expected.Raw, result.Raw)
			}
		})
	}
}

func TestVersionHistory_GetLatestVersion(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-time.Hour)

	tests := []struct {
		name     string
		history  *VersionHistory
		expected *Version
	}{
		{
			name: "empty version history",
			history: &VersionHistory{
				ComponentName: "test-component",
				Versions:      []Version{},
			},
			expected: nil,
		},
		{
			name: "single non-deprecated version",
			history: &VersionHistory{
				ComponentName: "test-component",
				Versions: []Version{
					{
						Version:    "1.0.0",
						CreatedAt:  now,
						Deprecated: false,
					},
				},
			},
			expected: &Version{
				Version:    "1.0.0",
				CreatedAt:  now,
				Deprecated: false,
			},
		},
		{
			name: "multiple versions with latest non-deprecated",
			history: &VersionHistory{
				ComponentName: "test-component",
				Versions: []Version{
					{
						Version:    "2.0.0",
						CreatedAt:  now,
						Deprecated: false,
					},
					{
						Version:    "1.0.0",
						CreatedAt:  earlier,
						Deprecated: false,
					},
				},
			},
			expected: &Version{
				Version:    "2.0.0",
				CreatedAt:  now,
				Deprecated: false,
			},
		},
		{
			name: "all versions deprecated - returns first (latest)",
			history: &VersionHistory{
				ComponentName: "test-component",
				Versions: []Version{
					{
						Version:    "2.0.0",
						CreatedAt:  now,
						Deprecated: true,
					},
					{
						Version:    "1.0.0",
						CreatedAt:  earlier,
						Deprecated: true,
					},
				},
			},
			expected: &Version{
				Version:    "2.0.0",
				CreatedAt:  now,
				Deprecated: true,
			},
		},
		{
			name: "mixed deprecated and non-deprecated - returns latest non-deprecated",
			history: &VersionHistory{
				ComponentName: "test-component",
				Versions: []Version{
					{
						Version:    "3.0.0",
						CreatedAt:  now,
						Deprecated: true,
					},
					{
						Version:    "2.0.0",
						CreatedAt:  earlier,
						Deprecated: false,
					},
					{
						Version:    "1.0.0",
						CreatedAt:  earlier.Add(-time.Hour),
						Deprecated: true,
					},
				},
			},
			expected: &Version{
				Version:    "2.0.0",
				CreatedAt:  earlier,
				Deprecated: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.history.GetLatestVersion()

			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Version, result.Version)
				assert.Equal(t, tt.expected.Deprecated, result.Deprecated)
				assert.Equal(t, tt.expected.CreatedAt, result.CreatedAt)
			}
		})
	}
}

func TestVersionHistory_GetLatestNonDeprecatedVersion(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-time.Hour)

	tests := []struct {
		name     string
		history  *VersionHistory
		expected *Version
	}{
		{
			name: "empty version history",
			history: &VersionHistory{
				ComponentName: "test-component",
				Versions:      []Version{},
			},
			expected: nil,
		},
		{
			name: "single non-deprecated version",
			history: &VersionHistory{
				ComponentName: "test-component",
				Versions: []Version{
					{
						Version:    "1.0.0",
						CreatedAt:  now,
						Deprecated: false,
					},
				},
			},
			expected: &Version{
				Version:    "1.0.0",
				CreatedAt:  now,
				Deprecated: false,
			},
		},
		{
			name: "all versions deprecated",
			history: &VersionHistory{
				ComponentName: "test-component",
				Versions: []Version{
					{
						Version:    "2.0.0",
						CreatedAt:  now,
						Deprecated: true,
					},
					{
						Version:    "1.0.0",
						CreatedAt:  earlier,
						Deprecated: true,
					},
				},
			},
			expected: nil,
		},
		{
			name: "mixed deprecated and non-deprecated",
			history: &VersionHistory{
				ComponentName: "test-component",
				Versions: []Version{
					{
						Version:    "3.0.0",
						CreatedAt:  now,
						Deprecated: true,
					},
					{
						Version:    "2.0.0",
						CreatedAt:  earlier,
						Deprecated: false,
					},
					{
						Version:    "1.0.0",
						CreatedAt:  earlier.Add(-time.Hour),
						Deprecated: false,
					},
				},
			},
			expected: &Version{
				Version:    "2.0.0",
				CreatedAt:  earlier,
				Deprecated: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.history.GetLatestNonDeprecatedVersion()

			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Version, result.Version)
				assert.Equal(t, tt.expected.Deprecated, result.Deprecated)
				assert.Equal(t, tt.expected.CreatedAt, result.CreatedAt)
			}
		})
	}
}
