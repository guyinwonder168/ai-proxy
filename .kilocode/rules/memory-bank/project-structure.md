# AI Proxy Server - Project Structure

**Last Updated:** 2025-08-23T20:19:56.487Z

## Project Overview

The AI Proxy Server is a Go-based HTTP proxy that provides a unified API interface for multiple AI providers. It implements intelligent routing, rate limiting, and extensible provider architecture with centralized HTTP client management.

## Directory Structure

```
ai-proxy/
├── .env                           # Environment variables file
├── .gitignore                     # Git ignore file
├── .golangci.yml                  # GolangCI-Lint configuration
├── ai-proxy                       # Compiled binary (output)
├── go.mod                         # Go module definition
├── go.sum                         # Go module checksums
├── LICENSE                        # Project license
├── main.go                        # Main entry point
├── provider_config.yaml           # Provider configuration
├── README.md                      # Project documentation
├── .github/                       # GitHub configuration
│   └── workflows/                 # GitHub Actions workflows
├── .kilocode/                     # KiloCode configuration
│   └── rules/                     # KiloCode rules
│       └── memory-bank/           # Memory bank files
│           ├── analysis.md        # Architecture analysis
│           ├── architecture.md    # System architecture
│           ├── brief.md           # Project brief
│           ├── context.md         # Current work context
│           ├── tech.md            # Technology stack
│           └── project-structure.md # This file
├── .vscode/                       # VSCode configuration
│   ├── extensions.json            # Recommended extensions
│   └── settings.json              # Workspace settings
├── internal/                      # Internal packages
│   ├── bufferpool/                # Buffer pooling implementation
│   │   └── pool.go                # Buffer pool implementation
│   ├── client/                    # HTTP client factory
│   │   └── http_client.go         # HTTP client creation and configuration
│   ├── config/                    # Configuration management
│   │   └── config.go              # Configuration loading and initialization
│   ├── errors/                    # Error handling utilities
│   │   └── errors.go              # Structured error types and handling
│   ├── handlers/                  # HTTP request handlers
│   │   ├── health.go              # Health check handler
│   │   ├── image.go               # Image processing handler
│   │   ├── models.go              # Models listing handler
│   │   └── proxy.go               # Main proxy handler implementation
│   ├── middleware/                # HTTP middleware
│   │   └── auth.go                # Authentication middleware
│   ├── provider/                  # AI provider adapters
│   │   ├── registry.go            # Provider registry
│   │   ├── cloudflare/            # Cloudflare provider adapter
│   │   │   └── client.go          # Cloudflare API client implementation
│   │   ├── gemini/                # Google Gemini provider adapter
│   │   │   └── client.go          # Gemini API client implementation
│   │   ├── gigachat/              # Sberbank GigaChat provider adapter
│   │   │   ├── client.go          # GigaChat API client implementation
│   │   │   └── convert.go         # GigaChat response conversion utilities
│   │   ├── groq/                  # Groq provider adapter
│   │   │   └── client.go          # Groq API client implementation
│   │   ├── openai/                # OpenAI provider adapter
│   │   │   └── client.go          # OpenAI API client implementation
│   │   ├── openrouter/            # OpenRouter provider adapter
│   │   │   ├── client.go          # OpenRouter API client implementation
│   │   │   ├── client_test.go     # OpenRouter client tests
│   │   │   ├── convert.go         # OpenRouter response conversion utilities
│   │   │   └── models.go          # OpenRouter model definitions
│   ├── schema/                    # Shared data structures
│   │   └── schema.go              # Provider interface and data structures
│   ├── server/                    # HTTP server implementation
│   │   ├── routes.go              # Route registration
│   │   └── server.go              # Server initialization and management
│   └── stream/                    # Streaming utilities
│       └── retry.go               # Streaming retry mechanism implementation
└── scripts/                       # Deployment and configuration scripts
    ├── .env.example               # Example environment variables file
    ├── Dockerfile                 # Docker container configuration
    └── provider_config-example.yaml # Example provider configuration
```

## File Descriptions

### Root Level Files

- **`.env`** - Environment variables file showing required configuration options
- **`.gitignore`** - Specifies intentionally untracked files to ignore in Git
- **`.golangci.yml`** - Configuration file for GolangCI-Lint static analysis tool
- **`ai-proxy`** - The compiled binary output of the application
- **`go.mod`** - Go module definition file listing dependencies
- **`go.sum`** - Go module checksums for dependency verification
- **`LICENSE`** - The project license file
- **`main.go`** - Main entry point of the application, initializes providers and starts HTTP server
- **`provider_config.yaml`** - Configuration file for AI providers
- **`README.md`** - Main project documentation with setup and usage instructions

### .github Directory

- **`workflows/`** - Contains GitHub Actions workflow definitions for CI/CD

### .kilocode Directory

- **`rules/`** - Contains KiloCode configuration rules
- **`rules/memory-bank/`** - Contains memory bank files for project documentation and context
  - **`analysis.md`** - Detailed architecture analysis and optimization findings
  - **`architecture.md`** - System architecture documentation with diagrams
  - **`brief.md`** - Project brief and core features overview
  - **`context.md`** - Current work focus and recent changes
  - **`tech.md`** - Technology stack and implementation details
  - **`project-structure.md`** - This file containing project structure documentation

### .vscode Directory

- **`extensions.json`** - Recommended VSCode extensions for the project
- **`settings.json`** - Workspace-specific VSCode settings

### internal Directory

The `internal` directory contains all the core implementation packages of the AI Proxy Server, organized with clear separation of concerns.

#### bufferpool Package

- **`pool.go`** - Implements buffer pooling with `sync.Pool` for efficient memory management, providing multiple buffer sizes (1K, 4K, 16K) for different payload sizes to reduce garbage collection pressure.

#### client Package

- **`http_client.go`** - Contains the HTTP client factory for creating configured HTTP clients with centralized client creation, configurable timeouts, and connection pooling.

#### config Package

- **`config.go`** - Handles configuration loading from YAML files, initialization of models and rate limits, and loading of retry configuration from environment variables.

#### errors Package

- **`errors.go`** - Implements structured error handling with typed errors, providing specific error types for different error conditions (RateLimitError, ProviderError, NetworkError, etc.) and retryable error detection for automatic recovery.

#### handlers Package

- **`health.go`** - Implements the health check endpoint for monitoring service availability.
- **`image.go`** - Handles image generation requests, routing them to appropriate providers and processing responses.
- **`models.go`** - Implements the models listing endpoint to show available models.
- **`proxy.go`** - Main proxy handler implementation containing the core logic for routing requests to appropriate providers, rate limiting, and response handling.

#### middleware Package

- **`auth.go`** - Implements authentication middleware for validating API requests with proper security headers and error handling.

#### provider Package

The provider package contains all AI provider adapters, each implementing the Provider interface.

- **`registry.go`** - Provider registry for managing available providers.
- **`cloudflare/`** - Cloudflare provider adapter implementing the Provider interface for Cloudflare's AI Gateway API.
- **`gemini/`** - Google Gemini provider adapter implementing the Provider interface for Google's Gemini API with proper mutex-based throttling.
- **`gigachat/`** - Sberbank GigaChat provider adapter using the official SDK with OAuth2 token rotation.
  - **`client.go`** - GigaChat API client implementation
  - **`convert.go`** - GigaChat response conversion utilities for translating API responses.
- **`groq/`** - Groq provider adapter implementing the Provider interface for Groq's API with proper mutex-based throttling.
- **`openai/`** - OpenAI provider adapter implementing the Provider interface for OpenAI-compatible APIs.
- **`openrouter/`** - OpenRouter provider adapter implementing the Provider interface for OpenRouter's API, including streaming support.
 - **`client.go`** - OpenRouter API client implementation
  - **`client_test.go`** - Tests for the OpenRouter client implementation.
  - **`convert.go`** - OpenRouter response conversion utilities for translating API responses.
  - **`models.go`** - OpenRouter model definitions and configurations.

#### schema Package

- **`schema.go`** - Defines shared data structures and interfaces, including the Provider interface contract with Call method for all AI providers, HTTP client configuration structures for provider settings, OpenAI-compatible request/response structures for standardization, and error handling structures for provider-specific errors.

#### server Package

- **`routes.go`** - Handles route registration for all HTTP endpoints, including authenticated and public routes.
- **`server.go`** - Implements HTTP server initialization, configuration loading from environment variables and files, and server lifecycle management.

#### stream Package

- **`retry.go`** - Implements streaming retry mechanism with exponential backoff and jitter, supporting stream recreation for retryable errors.

### scripts Directory

- **`.env.example`** - Example environment variables file showing required configuration options
- **`Dockerfile`** - Docker container configuration for containerized deployment
- **`provider_config-example.yaml`** - Example configuration file for AI providers

## Package Relationships

The internal packages are organized with clear separation of concerns:

1. **Server Layer** (`server/`) - HTTP server initialization, configuration, and route registration
2. **Middleware Layer** (`middleware/`) - HTTP middleware for authentication and other cross-cutting concerns
3. **Handler Layer** (`handlers/`) - HTTP request handlers for different endpoints
4. **Business Logic Layer** (`config/`, `bufferpool/`, `errors/`, `stream/`) - Core business logic, configuration management, error handling, and utility functions
5. **Provider Adapters** (`provider/`) - Implementations for each AI provider
6. **Shared Utilities** (`client/`, `schema/`) - Common functionality used across providers
7. **Configuration** (`provider_config.yaml`, `.env`) - Configuration files for setting up providers and environment variables

This structure allows for easy addition of new providers while maintaining a consistent interface and shared infrastructure. The clear separation of concerns makes the codebase more maintainable and testable.