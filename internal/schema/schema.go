package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"ai-proxy/internal/errors"
)

type RequestOpenAICompatable struct {
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages,omitempty"`
	Model string `json:"model,omitempty"`
}

type ResponseOpenAICompatable struct {
	ID      string `json:"id,omitempty"`
	Model   string `json:"model,omitempty"`
	Created int    `json:"created,omitempty"`
	Choices []struct {
		Index   int `json:"index,omitempty"`
		Message struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"message,omitempty"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices,omitempty"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens,omitempty"`
		CompletionTokens int `json:"completion_tokens,omitempty"`
		TotalTokens      int `json:"total_tokens,omitempty"`
	} `json:"usage,omitempty"`
}

// Provider interface defines the contract for all AI providers
type Provider interface {
	Call(ctx context.Context, payload []byte, path string) ([]byte, error)
}

// HttpClientConfig defines HTTP client settings for providers
type HttpClientConfig struct {
	TimeoutSeconds         int `yaml:"timeout_seconds" json:"timeout_seconds"`
	IdleConnTimeoutSeconds int `yaml:"idle_conn_timeout_seconds" json:"idle_conn_timeout_seconds"`
}

// OpenRouterError represents an error from the OpenRouter API
type OpenRouterError struct {
	StatusCode int
	Body       string
	Message    string
	Code       int
	Provider   string
}

// Error implements the error interface
func (e *OpenRouterError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("OpenRouter error: %s (status: %d)", e.Message, e.StatusCode)
	}
	return fmt.Sprintf("OpenRouter error (status: %d)", e.StatusCode)
}

// IsRetryable checks if the error is retryable
func (e *OpenRouterError) IsRetryable() bool {
	// 429 (rate limit) and 5xx errors are typically retryable
	return e.StatusCode == http.StatusTooManyRequests ||
		(e.StatusCode >= 500 && e.StatusCode < 600)
}

// ToAIProxyError converts an OpenRouterError to an AIProxyError
func (e *OpenRouterError) ToAIProxyError(modelName string) *errors.AIProxyError {
	if e.StatusCode == http.StatusTooManyRequests {
		return errors.NewRateLimitError(e.Message, modelName, e)
	}

	return errors.NewProviderError(e.Message, "OpenRouter", modelName, e.StatusCode, e)
}

// ParseOpenRouterError parses an error response from OpenRouter
func ParseOpenRouterError(statusCode int, body []byte) *OpenRouterError {
	errorObj := &OpenRouterError{
		StatusCode: statusCode,
		Body:       string(body),
	}

	// Try to parse the JSON error response
	var response struct {
		Error struct {
			Message  string `json:"message"`
			Code     int    `json:"code"`
			Metadata struct {
				ProviderName string `json:"provider_name"`
			} `json:"metadata"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &response); err == nil {
		errorObj.Message = response.Error.Message
		errorObj.Code = response.Error.Code
		errorObj.Provider = response.Error.Metadata.ProviderName
	}

	return errorObj
}
