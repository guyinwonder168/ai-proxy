package provider

import (
	"context"
	"fmt"

	"ai-proxy/internal/provider/cloudflare"
	"ai-proxy/internal/provider/gemini"
	"ai-proxy/internal/provider/gigachat"
	"ai-proxy/internal/provider/groq"
	"ai-proxy/internal/provider/openai"
	"ai-proxy/internal/provider/openrouter"
	"ai-proxy/internal/schema"
)

// ProviderConstructor defines a function type for creating providers
type ProviderConstructor func(providerURL, model, token string, config schema.HttpClientConfig) schema.Provider

// providerRegistry maps provider names to their constructor functions
var providerRegistry = map[string]ProviderConstructor{
	"cloudflare": func(providerURL, model, token string, config schema.HttpClientConfig) schema.Provider {
		return cloudflare.NewCloudflareClient(providerURL, model, token, config)
	},
	"google": func(providerURL, model, token string, config schema.HttpClientConfig) schema.Provider {
		return gemini.NewGeminiClient(providerURL, model, token, config)
	},
	"gigachat": func(providerURL, model, token string, config schema.HttpClientConfig) schema.Provider {
		// Note: Gigachat uses a different initialization pattern
		return &gigachat.GigaChatClient{
			ProviderURL: providerURL,
			Model:       model,
			Token:       token,
		}
	},
	"groq": func(providerURL, model, token string, config schema.HttpClientConfig) schema.Provider {
		return groq.NewGroqClient(providerURL, model, token, config)
	},
	"arliai": func(providerURL, model, token string, config schema.HttpClientConfig) schema.Provider {
		return groq.NewGroqClient(providerURL, model, token, config)
	},
	"github": func(providerURL, model, token string, config schema.HttpClientConfig) schema.Provider {
		return groq.NewGroqClient(providerURL, model, token, config)
	},
	"cohere": func(providerURL, model, token string, config schema.HttpClientConfig) schema.Provider {
		return openai.NewOpenAIClient(providerURL, model, token, config)
	},
	"openrouter": func(providerURL, model, token string, config schema.HttpClientConfig) schema.Provider {
		return openrouter.NewOpenRouterClient(providerURL, model, token, config)
	},
	// Default to OpenAI-compatible provider for any other provider types
	"default": func(providerURL, model, token string, config schema.HttpClientConfig) schema.Provider {
		return openai.NewOpenAIClient(providerURL, model, token, config)
	},
}

// GetProvider creates a new provider instance based on the provider name
func GetProvider(providerName, providerURL, model, token string, config schema.HttpClientConfig) (schema.Provider, error) {
	constructor, exists := providerRegistry[providerName]
	if !exists {
		// Use default provider for unknown provider types
		constructor = providerRegistry["default"]
	}

	// Special handling for gigachat provider
	if providerName == "gigachat" {
		provider := constructor(providerURL, model, token, config)
		// Gigachat requires special initialization
		if err := gigachat.InitClient(token); err != nil {
			return nil, fmt.Errorf("failed to initialize gigachat client: %w", err)
		}
		return provider, nil
	}

	return constructor(providerURL, model, token, config), nil
}

// RegisterProvider allows registering new provider constructors at runtime
func RegisterProvider(name string, constructor ProviderConstructor) {
	providerRegistry[name] = constructor
}

// GetSupportedProviders returns a list of all supported provider names
func GetSupportedProviders() []string {
	providers := make([]string, 0, len(providerRegistry))

	// Skip the "default" provider as it's not a real provider type
	for name := range providerRegistry {
		if name != "default" {
			providers = append(providers, name)
		}
	}

	return providers
}

// CallProvider is a convenience function that creates and calls a provider in one step
func CallProvider(ctx context.Context, providerName, providerURL, model, token string, config schema.HttpClientConfig, payload []byte, path string) ([]byte, error) {
	provider, err := GetProvider(providerName, providerURL, model, token, config)
	if err != nil {
		return nil, err
	}

	return provider.Call(ctx, payload, path)
}
