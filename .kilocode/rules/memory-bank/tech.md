## Technology Stack

**Core Infrastructure:**
- Go 1.24 (standard library net/http)
- Concurrent architecture with sync.RWMutex
- GJSON/SJSON for high-performance JSON manipulation
- Yaml.v3 for configuration management
- godotenv for environment variable loading
- Google UUID for unique identifier generation

**AI Providers:**
- Gigachat (official SDK with OAuth2 token rotation)
- Gemini (REST API integration)
- OpenAI (compatible API format)
- Groq (Cloud API)
- OpenRouter (API integration)
- Cloudflare (AI Gateway API)
- Cohere (OpenAI-compatible API)

**Key Implementations:**
- **Rate Limiting**:
  - Three-tier counters (minute/hour/day)
  - Mutex-protected concurrent access
  - Automatic time-window resets
- **HTTP Client Management**:
  - Centralized client factory with configurable timeouts
  - Connection pooling tuned via config.yaml
  - TLS settings and proxy support
  - Circuit breaker pattern for error handling
- **Provider Adapters**:
 - Standardized request/response translation
  - Automatic token management
- **Streaming**:
 - Utilizes `io.Copy` to efficiently pipe response bodies for streaming requests.
  - Serves chunked data using the `text/event-stream` content type for Server-Sent Events (SSE).
  - Implements retryable streaming with exponential backoff and jitter
- **Buffer Pooling**:
  - Uses `sync.Pool` for efficient memory management
  - Implements multiple buffer sizes (1K, 4K, 16K) for different payload sizes
  - Reduces garbage collection pressure by reusing byte slices and readers
- **Structured Error Handling**:
  - Implements typed errors with specific error codes and metadata
  - Provides retryable error detection for automatic recovery
  - Standardized error format across all providers
- **Streaming Retry Mechanism**:
  - Implements exponential backoff with jitter for streaming connections
  - Supports stream recreation for retryable errors
  - Configurable retry parameters via environment variables
- **Configuration Management**:
  - YAML-based configuration for models and providers
 - Environment variable support for sensitive data
  - Embedded configuration support
  - Retry configuration via environment variables
- **Authentication**:
  - Bearer token authentication middleware
  - Proper WWW-Authenticate headers
  - Secure error handling without information disclosure
- **Server Management**:
  - Graceful startup and shutdown
  - CLI flag support for configuration
  - Version information support

**Development Ecosystem:**
- GolangCI-Lint for code quality
- Go Modules dependency management
- VSCode with Go extension pack
- Built-in benchmarking/profiling
- Docker containerization support

**Current Dependencies:**
- github.com/joho/godotenv v1.5.1 (for environment variable loading)
- github.com/tidwall/gjson v1.18.0 (for JSON parsing)
- github.com/tidwall/sjson v1.2.5 (for JSON manipulation)
- gopkg.in/yaml.v3 v3.0.1 (for YAML parsing)
- github.com/google/uuid v1.6.0 (for UUID generation)
- github.com/evgensoft/gigachat (for Gigachat integration)

**Recently Identified Anti-patterns:**
- None - all previously identified anti-patterns have been addressed

**Implemented Technical Improvements:**
- Replaced channel-as-mutex patterns with proper sync.Mutex primitives in provider implementations
- Implemented dependency injection to eliminate global state
- Added buffer pooling with sync.Pool for memory efficiency in request/response handling
- Refactored rate limiting with proper locking for better concurrency
- Enhanced error handling with structured error types
- Improved context propagation for request cancellation
- Optimized struct field ordering for better memory alignment
- Implemented connection pooling enhancements with dynamic sizing (MaxIdleConns: 200, MaxIdleConnsPerHost: 50)
- Added streaming retry mechanism with exponential backoff and jitter
- Improved authentication middleware with proper security headers
- Added retry configuration via environment variables
- Implemented structured error handling with typed errors and retryable error detection
- Added buffer pooling for efficient memory management in both text and image handlers
- Enhanced streaming support with retryable streams for OpenRouter provider
- Fixed race conditions in rate limiting with proper mutex locking
- Eliminated global state through dependency injection patterns
- Improved memory management with buffer pooling across all handlers
- Enhanced error categorization with specific error types for different conditions
- Added streaming retry mechanisms with exponential backoff for better resilience
- Improved context propagation for request cancellation and timeout handling
- Implemented modular package structure with clear separation of concerns
- Added proper server initialization and lifecycle management
- Enhanced configuration management with environment variable support
- Improved route registration with authenticated and public endpoints

**Current Internal Package Structure:**
- **bufferpool**: Implements buffer pooling with `sync.Pool` for efficient memory management
  - Provides multiple buffer sizes (1K, 4K, 16K) for different payload sizes
  - Reduces garbage collection pressure by reusing byte slices and readers
  - Used in both proxy handler and provider clients for request/response handling
- **client**: Contains the HTTP client factory for creating configured HTTP clients
  - Centralized client creation with configurable timeouts and connection pooling
- **config**: Handles configuration loading from YAML files and environment variables
  - Initializes models and rate limits
  - Loads retry configuration from environment variables
  - Manages global configuration state
- **errors**: Implements structured error handling with typed errors
  - Provides specific error types for different error conditions (RateLimitError, ProviderError, NetworkError, etc.)
  - Implements retryable error detection for automatic recovery
  - Used throughout the codebase for consistent error handling
- **handlers**: Contains HTTP request handlers for different endpoints
  - Implements proxy handler for chat completions
  - Implements image handler for image generation requests
  - Implements models handler for listing available models
  - Implements health check handler for monitoring service availability
- **middleware**: Contains HTTP middleware functions
  - Implements authentication middleware for validating API requests<thinking>
I've updated the architecture.md file. Now I'll update the tech.md file to include the latest technology stack and improvements.
</thinking>

<write_to_file>
<path>.kilocode/rules/memory-bank/tech.md</path>
<content>
## Technology Stack

**Core Infrastructure:**
- Go 1.24 (standard library net/http)
- Concurrent architecture with sync.RWMutex
- GJSON/SJSON for high-performance JSON manipulation
- Yaml.v3 for configuration management
- godotenv for environment variable loading
- Google UUID for unique identifier generation

**AI Providers:**
- Gigachat (official SDK with OAuth2 token rotation)
- Gemini (REST API integration)
- OpenAI (compatible API format)
- Groq (Cloud API)
- OpenRouter (API integration)
- Cloudflare (AI Gateway API)
- Cohere (OpenAI-compatible API)

**Key Implementations:**
- **Rate Limiting**:
  - Three-tier counters (minute/hour/day)
  - Mutex-protected concurrent access
  - Automatic time-window resets
- **HTTP Client Management**:
  - Centralized client factory with configurable timeouts
  - Connection pooling tuned via config.yaml
  - TLS settings and proxy support
  - Circuit breaker pattern for error handling
- **Provider Adapters**:
 - Standardized request/response translation
  - Automatic token management
- **Streaming**:
 - Utilizes `io.Copy` to efficiently pipe response bodies for streaming requests.
  - Serves chunked data using the `text/event-stream` content type for Server-Sent Events (SSE).
  - Implements retryable streaming with exponential backoff and jitter
- **Buffer Pooling**:
  - Uses `sync.Pool` for efficient memory management
  - Implements multiple buffer sizes (1K, 4K, 16K) for different payload sizes
  - Reduces garbage collection pressure by reusing byte slices and readers
- **Structured Error Handling**:
  - Implements typed errors with specific error codes and metadata
  - Provides retryable error detection for automatic recovery
  - Standardized error format across all providers
- **Streaming Retry Mechanism**:
  - Implements exponential backoff with jitter for streaming connections
  - Supports stream recreation for retryable errors
  - Configurable retry parameters via environment variables
- **Configuration Management**:
  - YAML-based configuration for models and providers
 - Environment variable support for sensitive data
  - Embedded configuration support
  - Retry configuration via environment variables
- **Authentication**:
  - Bearer token authentication middleware
  - Proper WWW-Authenticate headers
  - Secure error handling without information disclosure
- **Server Management**:
  - Graceful startup and shutdown
  - CLI flag support for configuration
  - Version information support

**Development Ecosystem:**
- GolangCI-Lint for code quality
- Go Modules dependency management
- VSCode with Go extension pack
- Built-in benchmarking/profiling
- Docker containerization support

**Current Dependencies:**
- github.com/joho/godotenv v1.5.1 (for environment variable loading)
- github.com/tidwall/gjson v1.18.0 (for JSON parsing)
- github.com/tidwall/sjson v1.2.5 (for JSON manipulation)
- gopkg.in/yaml.v3 v3.0.1 (for YAML parsing)
- github.com/google/uuid v1.6.0 (for UUID generation)
- github.com/evgensoft/gigachat (for Gigachat integration)

**Recently Identified Anti-patterns:**
- None - all previously identified anti-patterns have been addressed

**Implemented Technical Improvements:**
- Replaced channel-as-mutex patterns with proper sync.Mutex primitives in provider implementations
- Implemented dependency injection to eliminate global state
- Added buffer pooling with sync.Pool for memory efficiency in request/response handling
- Refactored rate limiting with proper locking for better concurrency
- Enhanced error handling with structured error types
- Improved context propagation for request cancellation
- Optimized struct field ordering for better memory alignment
- Implemented connection pooling enhancements with dynamic sizing (MaxIdleConns: 200, MaxIdleConnsPerHost: 50)
- Added streaming retry mechanism with exponential backoff and jitter
- Improved authentication middleware with proper security headers
- Added retry configuration via environment variables
- Implemented structured error handling with typed errors and retryable error detection
- Added buffer pooling for efficient memory management in both text and image handlers
- Enhanced streaming support with retryable streams for OpenRouter provider
- Fixed race conditions in rate limiting with proper mutex locking
- Eliminated global state through dependency injection patterns
- Improved memory management with buffer pooling across all handlers
- Enhanced error categorization with specific error types for different conditions
- Added streaming retry mechanisms with exponential backoff for better resilience
- Improved context propagation for request cancellation and timeout handling
- Implemented modular package structure with clear separation of concerns
- Added proper server initialization and lifecycle management
- Enhanced configuration management with environment variable support
- Improved route registration with authenticated and public endpoints

**Current Internal Package Structure:**
- **bufferpool**: Implements buffer pooling with `sync.Pool` for efficient memory management
  - Provides multiple buffer sizes (1K, 4K, 16K) for different payload sizes
  - Reduces garbage collection pressure by reusing byte slices and readers
  - Used in both proxy handler and provider clients for request/response handling
- **client**: Contains the HTTP client factory for creating configured HTTP clients
  - Centralized client creation with configurable timeouts and connection pooling
- **config**: Handles configuration loading from YAML files and environment variables
  - Initializes models and rate limits
  - Loads retry configuration from environment variables
  - Manages global configuration state
- **errors**: Implements structured error handling with typed errors
  - Provides specific error types for different error conditions (RateLimitError, ProviderError, NetworkError, etc.)
  - Implements retryable error detection for automatic recovery
  - Used throughout the codebase for consistent error handling
- **handlers**: Contains HTTP request handlers for different endpoints
  - Implements proxy handler for chat completions
  - Implements image handler for image generation requests
  - Implements models handler for listing available models
  - Implements health check handler for monitoring service availability
- **middleware**: Contains HTTP middleware functions
  - Implements authentication middleware for validating API requests
 - Provides proper security headers and error handling
- **provider**: Contains AI provider adapters, each implementing the Provider interface
  - **registry.go**: Provider registry for managing available providers
  - **cloudflare/**: Cloudflare provider adapter implementing the Provider interface for Cloudflare's AI Gateway API
  - **gemini/**: Google Gemini provider adapter implementing the Provider interface for Google's Gemini API with proper mutex-based throttling
  - **gigachat/**: Sberbank GigaChat provider adapter using the official SDK with OAuth2 token rotation
    - **client.go**: GigaChat API client implementation
    - **convert.go**: GigaChat response conversion utilities for translating API responses
  - **groq/**: Groq provider adapter implementing the Provider interface for Groq's API with proper mutex-based throttling
  - **openai/**: OpenAI provider adapter implementing the Provider interface for OpenAI-compatible APIs
  - **openrouter/**: OpenRouter provider adapter implementing the Provider interface for OpenRouter's API, including streaming support
    - **client.go**: OpenRouter API client implementation
    - **client_test.go**: Tests for the OpenRouter client implementation
    - **convert.go**: OpenRouter response conversion utilities for translating API responses
    - **models.go**: OpenRouter model definitions and configurations
- **schema**: Contains shared data structures and interfaces
  - Defines the Provider interface contract with Call method for all AI providers
  - Contains HTTP client configuration structures for provider settings
 - Defines OpenAI-compatible request/response structures for standardization
  - Implements error handling structures for provider-specific errors
- **server**: Implements HTTP server initialization and route registration
  - **server.go**: HTTP server initialization, configuration loading, and lifecycle management
  - **routes.go**: Route registration for all HTTP endpoints, including authenticated and public routes
- **stream**: Implements streaming retry mechanism
  - Provides retryable streaming with exponential backoff and jitter
  - Supports stream recreation for retryable errors
  - Used by OpenRouter provider for resilient streaming connections

**Last Updated:** 2025-08-23T20:18:43.462Z