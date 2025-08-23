## AI Proxy Server

### Core Features
- **Unified API Endpoints**:
  - POST /chat/completions - Text generation with streaming support
  - POST /image - Image generation processing
  - GET /models - List available models
  - GET /ping - Health check endpoint
- **Intelligent Routing**:
  - Matches request length to model capabilities
  - Prioritizes models by cost/performance ratio
  - Implements circuit breaker pattern for failures
  - Automatic fallback mechanisms for provider failures
- **Advanced Rate Management**:
  - Three-tier limits (minute/hour/day) with fine-grained locking
  - Concurrent-safe counters with sync.Mutex
  - Automatic limit resets with time windows
  - Circuit breaker functionality for error recovery
- **Enhanced Security**:
  - Bearer token authentication with proper WWW-Authenticate headers
  - Secure error handling without information disclosure
  - Input validation and sanitization
  - Environment variable support for sensitive configuration
- **Extensible Architecture**:
  - Plugin-style provider integrations
  - Standardized request/response formats
  - Centralized configuration via YAML
  - Dependency injection for better testability

### Key Components
1. **Model Configuration** (YAML):
   - Provider credentials and API endpoints
   - Token limits and pricing tiers
   - Model capabilities metadata
   - HTTP client configuration per provider
2. **Rate Limiter**:
   - Distributed counters with mutex locks
   - Usage metrics collection
   - Priority-based quota allocation
   - Automatic recovery mechanisms
3. **Provider Adapters**:
   - Gemini: Handles Google's API specifics with proper throttling
   - Gigachat: Manages Sberbank's auth flow with OAuth2 token rotation
   - OpenAI: Standard API compatibility
   - Groq: Cloud API integration with throttling
   - OpenRouter: API integration with streaming support and retry mechanisms
   - Cloudflare: AI Gateway API integration
   - Cohere: OpenAI-compatible API
4. **Request Processor**:
   - Payload validation and content sanitization
   - Response standardization to OpenAI format
   - Buffer pooling for efficient memory management
   - Context-aware request handling with cancellation support
5. **Streaming Support**:
   - Server-Sent Events (SSE) for real-time responses
   - Retry mechanisms with exponential backoff and jitter
   - Direct response piping to minimize memory usage
   - Provider-specific streaming implementations
6. **Error Handling**:
   - Structured error types with specific error codes
   - Retryable error detection for automatic recovery
   - Standardized error format across all providers
   - Proper error wrapping and chaining for debugging