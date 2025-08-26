// Package server provides HTTP server initialization and route registration for the AI proxy service.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"ai-proxy/internal/config"
	"ai-proxy/internal/errors"
	"ai-proxy/internal/handlers"
	"ai-proxy/internal/middleware"

	"github.com/joho/godotenv"
)

// Server holds the configuration and HTTP server instance.
type Server struct {
	Port           int
	AuthMiddleware func(http.HandlerFunc) http.HandlerFunc
	Mux            *http.ServeMux
	httpServer     *http.Server
}

// ServerConfig holds the configuration for creating a new Server.
type ServerConfig struct {
	Port int
	Auth string
	Addr string
}

// NewServer creates a new Server instance with the provided configuration.
func NewServer(config *ServerConfig) (*Server, error) {
	// Validate port range
	if config.Port < 1 || config.Port > 65535 {
		proxyErr := errors.NewConfigurationError(fmt.Sprintf("Invalid PORT value: %d. Must be between 1 and 65535", config.Port), nil)
		return nil, proxyErr
	}

	// Validate auth token
	if config.Auth == "" {
		proxyErr := errors.NewConfigurationError("Missing AUTH_TOKEN in environment variables", nil)
		return nil, proxyErr
	}

	// Create the server instance
	server := &Server{
		Port:           config.Port,
		AuthMiddleware: middleware.AuthMiddleware(config.Auth),
		Mux:            http.NewServeMux(),
	}

	// Register routes
	server.RegisterRoutes()

	// Create the HTTP server
	addr := config.Addr
	if addr == "" {
		addr = fmt.Sprintf(":%d", config.Port)
	}

	server.httpServer = &http.Server{
		Addr:    addr,
		Handler: server.Mux,
	}

	return server, nil
}

// Start begins listening and serving HTTP requests.
func (s *Server) Start() error {
	log.Printf("Listening on port %d", s.Port)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		proxyErr := errors.NewNetworkError(fmt.Sprintf("Failed to start server: %v", err), err)
		return proxyErr
	}
	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// ReloadAuthMiddleware updates the authentication middleware with a new auth token.
func (s *Server) ReloadAuthMiddleware(newAuthToken string) {
	s.AuthMiddleware = middleware.AuthMiddleware(newAuthToken)
	// Re-register routes with the new middleware
	s.RegisterRoutes()
}

// NewServerFromConfig creates a new Server instance by loading configuration from files and environment variables.
func NewServerFromConfig(configPath, envFilePath, addr string) (*Server, error) {
	// Load .env file if specified or try default locations
	var envFile map[string]string
	var err error

	if envFilePath != "" {
		// Load from specified path
		envFile, err = godotenv.Read(envFilePath)
		if err != nil {
			return nil, fmt.Errorf("error loading .env file from %s: %v", envFilePath, err)
		}
	} else {
		// Try default locations
		// First try .env in current directory
		envFile, err = godotenv.Read()
		if err != nil {
			// Try /config/.env (Docker default)
			envFile, err = godotenv.Read("/config/.env")
			if err != nil {
				log.Println("No .env file found, using system environment variables")
				// Use system environment variables
				envFile = make(map[string]string)
				for _, env := range os.Environ() {
					pair := []string{}
					for i, s := range env {
						if s == '=' {
							pair = append(pair, env[:i], env[i+1:])
							break
						}
					}
					if len(pair) == 2 {
						envFile[pair[0]] = pair[1]
					}
				}
			}
		}
	}

	// Load retry configuration from environment variables
	retryMaxRetries, retryBaseDelay, retryMaxDelay, retryJitter := config.LoadRetryConfig(envFile)

	// Load provider config: CLI --config is required
	config, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	// Initialize models and rate limits
	if err := config.Initialize(); err != nil {
		proxyErr := errors.NewConfigurationError(
			fmt.Sprintf("Error initializing models: %v", err), err)
		return nil, proxyErr
	}

	// Pass retry configuration to internal package
	handlers.SetRetryConfig(retryMaxRetries, retryBaseDelay, retryMaxDelay, retryJitter)

	// Get port from environment variable with default to 8080
	portStr, exists := envFile["PORT"]
	if !exists || portStr == "" {
		portStr = "8080" // Default port
	}

	// Convert port to integer
	port, err := strconv.Atoi(portStr)
	if err != nil {
		proxyErr := errors.NewConfigurationError(fmt.Sprintf("Invalid PORT value: %s. Must be a number", portStr), err)
		return nil, proxyErr
	}

	// Get auth token from environment
	authToken, exists := envFile["AUTH_TOKEN"]
	if !exists || authToken == "" {
		proxyErr := errors.NewConfigurationError("Missing AUTH_TOKEN in environment variables", nil)
		return nil, proxyErr
	}

	// Create server configuration
	serverConfig := &ServerConfig{
		Port: port,
		Auth: authToken,
		Addr: addr,
	}

	// Create and return the server
	return NewServer(serverConfig)
}
