package openrouter

import (
	"testing"

	"ai-proxy/internal/schema"
)

func TestParseOpenRouterError(t *testing.T) {
	// Test case 1: Valid OpenRouter error response with 429 status
	errorBody := []byte(`{"error":{"message":"Provider returned error","code":429,"metadata":{"raw":"qwen/qwen3-coder:free is temporarily rate-limited upstream. Please retry shortly, or add your own key to accumulate your rate limits: https://openrouter.ai/settings/integrations","provider_name":"Chutes"}},"user_id":"user_30vFwYf3qlvprxam2pA4FoDYlAR"}`)

	openRouterErr := schema.ParseOpenRouterError(429, errorBody)

	if openRouterErr.StatusCode != 429 {
		t.Errorf("Expected status code 429, got %d", openRouterErr.StatusCode)
	}

	if openRouterErr.Message != "Provider returned error" {
		t.Errorf("Expected message 'Provider returned error', got '%s'", openRouterErr.Message)
	}

	if openRouterErr.Code != 429 {
		t.Errorf("Expected code 429, got %d", openRouterErr.Code)
	}

	if openRouterErr.Provider != "Chutes" {
		t.Errorf("Expected provider 'Chutes', got '%s'", openRouterErr.Provider)
	}

	// Test if the error is correctly identified as retryable
	if !openRouterErr.IsRetryable() {
		t.Error("Expected error to be retryable")
	}

	// Test case 2: Non-retryable error (400)
	errorBody2 := []byte(`{"error":{"message":"Invalid request","code":400,"metadata":{"provider_name":"ProviderX"}}}`)
	openRouterErr2 := schema.ParseOpenRouterError(400, errorBody2)

	if openRouterErr2.IsRetryable() {
		t.Error("Expected error to be non-retryable")
	}

	// Test case 3: Server error (50) should be retryable
	errorBody3 := []byte(`{"error":{"message":"Internal server error","code":500,"metadata":{"provider_name":"ProviderX"}}}`)
	openRouterErr3 := schema.ParseOpenRouterError(500, errorBody3)

	if !openRouterErr3.IsRetryable() {
		t.Error("Expected error to be retryable")
	}

	// Test case 4: Invalid JSON should still create an error object
	invalidBody := []byte(`invalid json`)
	openRouterErr4 := schema.ParseOpenRouterError(429, invalidBody)

	if openRouterErr4.StatusCode != 429 {
		t.Errorf("Expected status code 429, got %d", openRouterErr4.StatusCode)
	}

	// Should have empty message when JSON is invalid
	if openRouterErr4.Message != "" {
		t.Errorf("Expected empty message for invalid JSON, got '%s'", openRouterErr4.Message)
	}
}
