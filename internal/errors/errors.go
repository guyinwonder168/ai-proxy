package errors

import (
	"fmt"
	"net/http"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// RateLimitErrorType represents a rate limit error
	RateLimitErrorType ErrorType = "rate_limit_error"

	// ProviderErrorType represents a provider error
	ProviderErrorType ErrorType = "provider_error"

	// NetworkErrorType represents a network error
	NetworkErrorType ErrorType = "network_error"

	// ConfigurationErrorType represents a configuration error
	ConfigurationErrorType ErrorType = "configuration_error"

	// ModelSelectionErrorType represents a model selection error
	ModelSelectionErrorType ErrorType = "model_selection_error"

	// ValidationErrorType represents a validation error
	ValidationErrorType ErrorType = "validation_error"

	// AuthenticationErrorType represents an authentication error
	AuthenticationErrorType ErrorType = "authentication_error"
)

// AIProxyError represents a structured error in the AI proxy
type AIProxyError struct {
	Type     ErrorType `json:"type"`
	Code     int       `json:"code"`
	Message  string    `json:"message"`
	Provider string    `json:"provider,omitempty"`
	Model    string    `json:"model,omitempty"`
	Cause    error     `json:"-"` // Not serialized, used for error chaining
}

// Error implements the error interface
func (e *AIProxyError) Error() string {
	if e.Provider != "" {
		return fmt.Sprintf("[%s] %s (provider: %s)", e.Type, e.Message, e.Provider)
	}
	if e.Model != "" {
		return fmt.Sprintf("[%s] %s (model: %s)", e.Type, e.Message, e.Model)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap implements the error unwrapping interface
func (e *AIProxyError) Unwrap() error {
	return e.Cause
}

// IsRateLimitError checks if the error is a rate limit error
func (e *AIProxyError) IsRateLimitError() bool {
	return e.Type == RateLimitErrorType
}

// IsProviderError checks if the error is a provider error
func (e *AIProxyError) IsProviderError() bool {
	return e.Type == ProviderErrorType
}

// IsNetworkError checks if the error is a network error
func (e *AIProxyError) IsNetworkError() bool {
	return e.Type == NetworkErrorType
}

// IsRetryable checks if the error is retryable
func (e *AIProxyError) IsRetryable() bool {
	// Rate limit errors and 5xx errors are typically retryable
	return e.Type == RateLimitErrorType ||
		(e.Code >= 500 && e.Code < 600) ||
		e.Code == http.StatusTooManyRequests
}

// NewRateLimitError creates a new rate limit error
func NewRateLimitError(message string, model string, cause error) *AIProxyError {
	return &AIProxyError{
		Type:    RateLimitErrorType,
		Code:    http.StatusTooManyRequests,
		Message: message,
		Model:   model,
		Cause:   cause,
	}
}

// NewProviderError creates a new provider error
func NewProviderError(message string, provider string, model string, code int, cause error) *AIProxyError {
	return &AIProxyError{
		Type:     ProviderErrorType,
		Code:     code,
		Message:  message,
		Provider: provider,
		Model:    model,
		Cause:    cause,
	}
}

// NewNetworkError creates a new network error
func NewNetworkError(message string, cause error) *AIProxyError {
	return &AIProxyError{
		Type:    NetworkErrorType,
		Code:    http.StatusBadGateway,
		Message: message,
		Cause:   cause,
	}
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(message string, cause error) *AIProxyError {
	return &AIProxyError{
		Type:    ConfigurationErrorType,
		Code:    http.StatusInternalServerError,
		Message: message,
		Cause:   cause,
	}
}

// NewModelSelectionError creates a new model selection error
func NewModelSelectionError(message string, cause error) *AIProxyError {
	return &AIProxyError{
		Type:    ModelSelectionErrorType,
		Code:    http.StatusServiceUnavailable,
		Message: message,
		Cause:   cause,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *AIProxyError {
	return &AIProxyError{
		Type:    ValidationErrorType,
		Code:    http.StatusBadRequest,
		Message: message,
	}
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(message string) *AIProxyError {
	return &AIProxyError{
		Type:    AuthenticationErrorType,
		Code:    http.StatusUnauthorized,
		Message: message,
	}
}

// IsAIProxyError checks if an error is an AIProxyError
func IsAIProxyError(err error) bool {
	_, ok := err.(*AIProxyError)
	return ok
}

// AsAIProxyError attempts to convert an error to an AIProxyError
func AsAIProxyError(err error) (*AIProxyError, bool) {
	if proxyErr, ok := err.(*AIProxyError); ok {
		return proxyErr, true
	}
	return nil, false
}
