package client

import (
	"net/http"
	"time"

	"ai-proxy/internal/schema"
)

// NewHttpClient creates a configured HTTP client based on the provided configuration
func NewHttpClient(config schema.HttpClientConfig) *http.Client {
	// Create a custom transport with optimized settings for high-concurrency scenarios
	transport := &http.Transport{
		// Connection pool settings optimized for high-concurrency:
		// MaxIdleConns: Total number of idle connections to keep across all hosts
		// MaxIdleConnsPerHost: Number of idle connections to keep per host
		// MaxConnsPerHost: Maximum number of connections per host (prevents connection overload)
		MaxIdleConns:          200, // Increased total idle connections
		MaxIdleConnsPerHost:   50,  // Increased idle connections per host
		MaxConnsPerHost:       100, // Added maximum connections per host
		IdleConnTimeout:       time.Duration(config.IdleConnTimeoutSeconds) * time.Second,
		ExpectContinueTimeout: 1 * time.Second, // Added expect continue timeout
	}

	// Create and configure the HTTP client
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(config.TimeoutSeconds) * time.Second,
	}

	return client
}
