package storage

import (
	"fmt"
)

// StorageError is the base error type for all storage-related errors
type StorageError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *StorageError) Error() string {
	return e.Message
}

// WithDetail adds a detail to the error
func (e *StorageError) WithDetail(key string, value any) *StorageError {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// NewStorageError creates a new StorageError
func NewStorageError(code, message string) *StorageError {
	return &StorageError{
		Code:    code,
		Message: message,
		Details: make(map[string]any),
	}
}

// Specific error types that implement the error interface
// These can be used with type assertions for more type-safe error handling

// ResourceNotFoundError indicates a requested resource was not found
type ResourceNotFoundError struct {
	*StorageError
	ResourceType string
	ResourceID   string
}

func NewResourceNotFoundError(resourceType, resourceID string) *ResourceNotFoundError {
	return &ResourceNotFoundError{
		StorageError: NewStorageError(
			"RESOURCE_NOT_FOUND",
			fmt.Sprintf("%s not found: %s", resourceType, resourceID),
		),
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}
}

// ComponentNotFoundError is a specific type of ResourceNotFoundError
type ComponentNotFoundError struct {
	*ResourceNotFoundError
	Name    string
	Version string
}

func NewComponentNotFoundError(name, version string) *ComponentNotFoundError {
	id := name
	if version != "" {
		id = fmt.Sprintf("%s:%s", name, version)
	}
	return &ComponentNotFoundError{
		ResourceNotFoundError: NewResourceNotFoundError("component", id),
		Name:                  name,
		Version:               version,
	}
}

// VersionNotFoundError is a specific type of ResourceNotFoundError
type VersionNotFoundError struct {
	*ResourceNotFoundError
	ComponentName string
	Version       string
}

func NewVersionNotFoundError(componentName, version string) *VersionNotFoundError {
	return &VersionNotFoundError{
		ResourceNotFoundError: NewResourceNotFoundError(
			"version",
			fmt.Sprintf("%s:%s", componentName, version),
		),
		ComponentName: componentName,
		Version:       version,
	}
}

// ResourceExistsError indicates a resource already exists
type ResourceExistsError struct {
	*StorageError
	ResourceType string
	ResourceID   string
}

func NewResourceExistsError(resourceType, resourceID string) *ResourceExistsError {
	return &ResourceExistsError{
		StorageError: NewStorageError(
			"RESOURCE_EXISTS",
			fmt.Sprintf("%s already exists: %s", resourceType, resourceID),
		),
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}
}

// ComponentExistsError is a specific type of ResourceExistsError
type ComponentExistsError struct {
	*ResourceExistsError
	Name    string
	Version string
}

func NewComponentExistsError(name, version string) *ComponentExistsError {
	return &ComponentExistsError{
		ResourceExistsError: NewResourceExistsError(
			"component",
			fmt.Sprintf("%s:%s", name, version),
		),
		Name:    name,
		Version: version,
	}
}

// ValidationError indicates invalid input
type ValidationError struct {
	*StorageError
	Field  string
	Reason string
}

func NewValidationError(field, reason string) *ValidationError {
	return &ValidationError{
		StorageError: NewStorageError(
			"VALIDATION_ERROR",
			fmt.Sprintf("validation error for %s: %s", field, reason),
		),
		Field:  field,
		Reason: reason,
	}
}

// StorageUnavailableError indicates the storage backend is unavailable
type StorageUnavailableError struct {
	*StorageError
	Reason string
}

func NewStorageUnavailableError(reason string) *StorageUnavailableError {
	return &StorageUnavailableError{
		StorageError: NewStorageError(
			"STORAGE_UNAVAILABLE",
			fmt.Sprintf("storage backend unavailable: %s", reason),
		),
		Reason: reason,
	}
}

// ThrottledError indicates the request was throttled
type ThrottledError struct {
	*StorageUnavailableError
}

func NewThrottledError(reason string) *ThrottledError {
	return &ThrottledError{
		StorageUnavailableError: &StorageUnavailableError{
			StorageError: NewStorageError(
				"THROTTLED",
				fmt.Sprintf("request throttled: %s", reason),
			),
			Reason: reason,
		},
	}
}

// ConfigurationError indicates an invalid configuration
type ConfigurationError struct {
	*StorageError
	ConfigItem string
}

func NewConfigurationError(configItem, reason string) *ConfigurationError {
	return &ConfigurationError{
		StorageError: NewStorageError(
			"CONFIGURATION_ERROR",
			fmt.Sprintf("invalid configuration for %s: %s", configItem, reason),
		),
		ConfigItem: configItem,
	}
}

// Common error instances for convenience
var (
	ErrComponentNotFound      = NewStorageError("COMPONENT_NOT_FOUND", "component not found")
	ErrVersionNotFound        = NewStorageError("VERSION_NOT_FOUND", "component version not found")
	ErrComponentExists        = NewStorageError("COMPONENT_EXISTS", "component already exists")
	ErrInvalidVersion         = NewStorageError("INVALID_VERSION", "invalid semantic version")
	ErrInvalidDateRange       = NewStorageError("INVALID_DATE_RANGE", "start date must be before end date")
	ErrEmptySearchQuery       = NewStorageError("EMPTY_SEARCH_QUERY", "search query cannot be empty")
	ErrUnsupportedStorageType = NewStorageError("UNSUPPORTED_STORAGE_TYPE", "unsupported storage type")
	ErrStorageNotAvailable    = NewStorageError("STORAGE_NOT_AVAILABLE", "storage backend not available")
	ErrInvalidInput           = NewStorageError("INVALID_INPUT", "invalid input provided")
	ErrInvalidConfig          = NewStorageError("INVALID_CONFIG", "invalid storage configuration")
)
