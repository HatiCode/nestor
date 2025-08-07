package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSemanticVersion(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expected    *SemanticVersionInfo
		expectError bool
	}{
		{
			name:    "basic version",
			version: "1.2.3",
			expected: &SemanticVersionInfo{
				Major: 1,
				Minor: 2,
				Patch: 3,
				Raw:   "1.2.3",
			},
			expectError: false,
		},
		{
			name:    "version with v prefix",
			version: "v1.2.3",
			expected: &SemanticVersionInfo{
				Major: 1,
				Minor: 2,
				Patch: 3,
				Raw:   "v1.2.3",
			},
			expectError: false,
		},
		{
			name:    "version with pre-release",
			version: "1.2.3-alpha",
			expected: &SemanticVersionInfo{
				Major:      1,
				Minor:      2,
				Patch:      3,
				PreRelease: "alpha",
				Raw:        "1.2.3-alpha",
			},
			expectError: false,
		},
		{
			name:    "version with complex pre-release",
			version: "1.2.3-alpha.1.beta",
			expected: &SemanticVersionInfo{
				Major:      1,
				Minor:      2,
				Patch:      3,
				PreRelease: "alpha.1.beta",
				Raw:        "1.2.3-alpha.1.beta",
			},
			expectError: false,
		},
		{
			name:    "version with build metadata",
			version: "1.2.3+build.1",
			expected: &SemanticVersionInfo{
				Major: 1,
				Minor: 2,
				Patch: 3,
				Build: "build.1",
				Raw:   "1.2.3+build.1",
			},
			expectError: false,
		},
		{
			name:    "version with pre-release and build",
			version: "1.2.3-alpha+build.1",
			expected: &SemanticVersionInfo{
				Major:      1,
				Minor:      2,
				Patch:      3,
				PreRelease: "alpha",
				Build:      "build.1",
				Raw:        "1.2.3-alpha+build.1",
			},
			expectError: false,
		},
		{
			name:        "invalid format - too few parts",
			version:     "1.2",
			expectError: true,
		},
		{
			name:        "invalid format - too many parts",
			version:     "1.2.3.4",
			expectError: true,
		},
		{
			name:        "invalid major version",
			version:     "a.2.3",
			expectError: true,
		},
		{
			name:        "invalid minor version",
			version:     "1.b.3",
			expectError: true,
		},
		{
			name:        "invalid patch version",
			version:     "1.2.c",
			expectError: true,
		},
		{
			name:        "empty version",
			version:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSemanticVersion(tt.version)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Major, result.Major)
				assert.Equal(t, tt.expected.Minor, result.Minor)
				assert.Equal(t, tt.expected.Patch, result.Patch)
				assert.Equal(t, tt.expected.PreRelease, result.PreRelease)
				assert.Equal(t, tt.expected.Build, result.Build)
				assert.Equal(t, tt.expected.Raw, result.Raw)
			}
		})
	}
}

func TestSemanticVersionInfo_String(t *testing.T) {
	tests := []struct {
		name     string
		version  *SemanticVersionInfo
		expected string
	}{
		{
			name: "basic version",
			version: &SemanticVersionInfo{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
			expected: "1.2.3",
		},
		{
			name: "version with pre-release",
			version: &SemanticVersionInfo{
				Major:      1,
				Minor:      2,
				Patch:      3,
				PreRelease: "alpha",
			},
			expected: "1.2.3-alpha",
		},
		{
			name: "version with build",
			version: &SemanticVersionInfo{
				Major: 1,
				Minor: 2,
				Patch: 3,
				Build: "build.1",
			},
			expected: "1.2.3=build.1",
		},
		{
			name: "version with pre-release and build",
			version: &SemanticVersionInfo{
				Major:      1,
				Minor:      2,
				Patch:      3,
				PreRelease: "alpha",
				Build:      "build.1",
			},
			expected: "1.2.3-alpha=build.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSemanticVersionInfo_Compare(t *testing.T) {
	tests := []struct {
		name     string
		version1 *SemanticVersionInfo
		version2 *SemanticVersionInfo
		expected int
	}{
		{
			name:     "equal versions",
			version1: &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3},
			version2: &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3},
			expected: 0,
		},
		{
			name:     "version1 major greater",
			version1: &SemanticVersionInfo{Major: 2, Minor: 0, Patch: 0},
			version2: &SemanticVersionInfo{Major: 1, Minor: 9, Patch: 9},
			expected: 1,
		},
		{
			name:     "version1 major less",
			version1: &SemanticVersionInfo{Major: 1, Minor: 9, Patch: 9},
			version2: &SemanticVersionInfo{Major: 2, Minor: 0, Patch: 0},
			expected: -1,
		},
		{
			name:     "version1 minor greater",
			version1: &SemanticVersionInfo{Major: 1, Minor: 3, Patch: 0},
			version2: &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 9},
			expected: 1,
		},
		{
			name:     "version1 minor less",
			version1: &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 9},
			version2: &SemanticVersionInfo{Major: 1, Minor: 3, Patch: 0},
			expected: -1,
		},
		{
			name:     "version1 patch greater",
			version1: &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 4},
			version2: &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3},
			expected: 1,
		},
		{
			name:     "version1 patch less",
			version1: &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3},
			version2: &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 4},
			expected: -1,
		},
		{
			name:     "stable version greater than pre-release",
			version1: &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3},
			version2: &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3, PreRelease: "alpha"},
			expected: 1,
		},
		{
			name:     "pre-release less than stable version",
			version1: &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3, PreRelease: "alpha"},
			version2: &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3},
			expected: -1,
		},
		{
			name:     "pre-release comparison",
			version1: &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3, PreRelease: "beta"},
			version2: &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3, PreRelease: "alpha"},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version1.Compare(tt.version2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSemanticVersionInfo_IsPreRelease(t *testing.T) {
	tests := []struct {
		name     string
		version  *SemanticVersionInfo
		expected bool
	}{
		{
			name:     "stable version",
			version:  &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3},
			expected: false,
		},
		{
			name:     "pre-release version",
			version:  &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3, PreRelease: "alpha"},
			expected: true,
		},
		{
			name:     "empty pre-release",
			version:  &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3, PreRelease: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version.IsPreRelease()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSemanticVersionInfo_IsStable(t *testing.T) {
	tests := []struct {
		name     string
		version  *SemanticVersionInfo
		expected bool
	}{
		{
			name:     "stable version",
			version:  &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3},
			expected: true,
		},
		{
			name:     "pre-release version",
			version:  &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3, PreRelease: "alpha"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version.IsStable()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSemanticVersionInfo_NextVersions(t *testing.T) {
	version := &SemanticVersionInfo{Major: 1, Minor: 2, Patch: 3}

	t.Run("NextMajor", func(t *testing.T) {
		next := version.NextMajor()
		assert.Equal(t, 2, next.Major)
		assert.Equal(t, 0, next.Minor)
		assert.Equal(t, 0, next.Patch)
		assert.Equal(t, "", next.PreRelease)
		assert.Equal(t, "", next.Build)
		assert.Equal(t, "2.0.0", next.Raw)
	})

	t.Run("NextMinor", func(t *testing.T) {
		next := version.NextMinor()
		assert.Equal(t, 1, next.Major)
		assert.Equal(t, 3, next.Minor)
		assert.Equal(t, 0, next.Patch)
		assert.Equal(t, "", next.PreRelease)
		assert.Equal(t, "", next.Build)
		assert.Equal(t, "1.3.0", next.Raw)
	})

	t.Run("NextPatch", func(t *testing.T) {
		next := version.NextPatch()
		assert.Equal(t, 1, next.Major)
		assert.Equal(t, 2, next.Minor)
		assert.Equal(t, 4, next.Patch)
		assert.Equal(t, "", next.PreRelease)
		assert.Equal(t, "", next.Build)
		assert.Equal(t, "1.2.4", next.Raw)
	})
}
