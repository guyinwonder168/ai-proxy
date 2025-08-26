[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=guyinwonder168_ai-proxy&metric=bugs)](https://sonarcloud.io/summary/new_code?id=guyinwonder168_ai-proxy)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=guyinwonder168_ai-proxy&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=guyinwonder168_ai-proxy)

# Proxy for LLM Providers (Fork)

> ⚠️ **This is a fork** of the original [ai-proxy](https://github.com/evgensoft/ai-proxy) project by [Evgeny Shepelyuk](https://github.com/evgensoft). This fork includes significant architectural improvements, enhanced security features, and performance optimizations.

## Overview
This service acts as a proxy for various Large Language Model (LLM) providers, including OpenAI, Groq, HuggingFace, and others. It allows users to seamlessly interact with multiple LLMs through a unified API, simplifying integration and request management with secure proxy access token authentication.

## Features
- **Unified API**: Work with multiple LLM providers using a single API endpoint.
- **Provider Flexibility**: Easily switch between different LLM providers without changing your application code.
- **Secure Authentication**: Proxy access token authentication for API security.
- **Request Management**: Handles routing and error handling.
- **Rate Limiting**: Supports per-model request limits (minute/hour/day).
- **Simple Configuration**: YAML-based setup with support for multiple models.
- **Streaming Support**: Supports streaming responses for providers that implement this feature.
- **Enhanced Error Handling**: Structured error types with specific error codes and metadata.
- **Memory Efficiency**: Buffer pooling for efficient memory management.
- **Retry Mechanisms**: Streaming retry mechanism with exponential backoff and jitter.
- **Modular Architecture**: Clean separation of concerns with dedicated packages for each functionality.

## Major Improvements in This Fork

This fork includes several significant improvements over the original project:

### Architecture
- **Modular Package Structure**: Clear separation of concerns with dedicated packages for server, handlers, middleware, config, errors, etc.
- **Dependency Injection**: Eliminated global state through proper dependency injection patterns.
- **Provider Registry**: Centralized provider management with runtime registration capability.
- **Buffer Pooling**: Implemented buffer pooling with `sync.Pool` for efficient memory management.

### Performance
- **Connection Pooling**: Enhanced HTTP client configuration with optimized connection pooling settings.
- **Memory Management**: Improved memory management with buffer pooling across all handlers.
- **Streaming Retry**: Implemented streaming retry mechanism with exponential backoff and jitter.
- **Structured Error Handling**: Enhanced error handling with structured error types and retryable error detection.

### Security
- **Secure Authentication**: Improved authentication middleware with proper security headers.
- **Configuration Management**: Enhanced configuration management with environment variable support.
- **Error Handling**: Secure error handling without information disclosure.

### Extensibility
- **Provider Interface**: Standardized provider interface contract with `Call` method for all AI providers.
- **HTTP Client Factory**: Centralized HTTP client creation with configurable timeouts and connection pooling.
- **Rate Limiting**: Improved rate limiting implementation with fine-grained locking.

## Getting Started

### Prerequisites
- Go (version 1.24 or higher)

### Installation

Clone the repository:
```bash
git clone https://github.com/your-username/ai-proxy.git
cd ai-proxy
```

### Configuration

- `.env` - Proxy settings (proxy access token, port)
  - Copy from `scripts/.env.example`
  - Set `AUTH_TOKEN` to a secure proxy access token for authenticating requests to this proxy
- `provider-config.yaml` - Provider-specific configurations
  - Configuration file is required
  - Use `--config` CLI flag to specify path
  - Add API keys for each provider

> ✅ You can list multiple models from different providers in the same file.
> 🛡️ Sensitive values like API tokens should be stored securely.

```yaml
models:
  - name: deepseek/deepseek-r1:free
    provider: openrouter
    priority: 1
    requests_per_minute: 20
    requests_per_hour: 1000
    requests_per_day: 1000
    url: "https://openrouter.ai/api/v1/chat/completions"
    token: "your_openrouter_api_token"
    max_request_length: 167731
    model_size: BIG
    http_client_config:
      timeout_seconds: 30
      idle_conn_timeout_seconds: 90
```

> ✅ You can list multiple models from different providers in the same file.
> 🛡️ Sensitive values like API tokens should be stored securely.

### Quick Start

1. Copy configuration templates:
```bash
cp scripts/.env.example .env
```

2. Edit `.env` and set:
```bash
AUTH_TOKEN="your_token_here"  # REQUIRED
PORT="8080"                   # Optional (default: 8080)
```

3. (Optional) Create and edit `provider-config.yaml` with your provider API keys

4. Build and run:
```bash
go build -o ai-proxy && ./ai-proxy
```

### Running the Service

To start the proxy server, run:

```bash
go run main.go
```
The server will start on `http://localhost:8080` by default. You can change the port by setting the `PORT` environment variable:
```bash
PORT=9090 go run main.go
```
Replace `9090` with your desired port number.

### CLI Flags

The service supports several CLI flags for configuration:

```bash
go run main.go --help

# Available flags:
# -addr string
#       listen address override, e.g., :8080
# -config string
#       path to provider-config.yaml (required)
# -env-file string
#       path to .env file
# -version
#       print version and exit
```

### Docker Support

The project includes Docker support for easy deployment:

```bash
# Build the Docker image
docker build -t ai-proxy -f scripts/Dockerfile .

# Run the container
docker run -d \
  --name ai-proxy \
  -p 8080:80 \
  -v $(pwd)/config:/config \
  ai-proxy
```

## Example Usage

To use the proxy, you need to include the proxy access token in the Authorization header of your requests:

### Using cURL

```bash
curl -X POST http://localhost:8080/chat/completions \
     -H "Authorization: Bearer ${AUTH_TOKEN}" \
     -H "Content-Type: application/json" \
     -d '{
           "model": "deepseek/deepseek-r1:free",
           "messages": [
             {
               "role": "system",
               "content": "You are a helpful assistant."
             },
             {
               "role": "user",
               "content": "Tell me a joke."
             }
           ]
         }'
```

Replace `${AUTH_TOKEN}` with your actual proxy access token.

## Streaming API

The proxy supports streaming responses for providers that implement this feature. To use streaming, clients must send `"stream": true` in the JSON body of their `POST` request to `/chat/completions`. You also need to include the proxy access token in the Authorization header.

Here's an example of a streaming request using `curl`:

```bash
curl -N -X POST http://localhost:8080/chat/completions \
-H "Authorization: Bearer ${AUTH_TOKEN}" \
-H "Content-Type: application/json" \
-d '{
  "model": "deepseek/deepseek-r1:free",
  "messages": [
    {
      "role": "user",
      "content": "Tell me a very long story."
    }
  ],
  "stream": true
}'
```

Replace `${AUTH_TOKEN}` with your actual proxy access token.

When streaming is enabled, the response will be a `text/event-stream` containing server-sent events (SSE). Each event will contain a chunk of the response as it's generated by the model.

## Supported Providers

The proxy currently supports the following AI providers:

- **OpenRouter** - Access to multiple models including DeepSeek, Qwen, and others
- **Cloudflare** - Cloudflare AI Gateway
- **Google Gemini** - Google's Gemini models
- **Sberbank GigaChat** - Russian LLM provider
- **Groq** - Fast inference models
- **OpenAI** - OpenAI models (GPT-3.5, GPT-4, etc.)
- **Cohere** - Cohere models

## API Endpoints

The proxy exposes the following API endpoints:

- `POST /chat/completions` - Text generation with streaming support
- `POST /image` - Image generation processing (not fully implemented)
- `GET /models` - List available models
- `GET /ping` - Health check endpoint

## Contributing

Contributions are welcome! Please submit a pull request or open an issue to discuss improvements.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Original project by [Evgeny Shepelyuk](https://github.com/evgensoft)
- This fork contains significant improvements and modifications
