// Package server provides HTTP server initialization and route registration for the AI proxy service.
package server

import (
	"ai-proxy/internal/handlers"
)

// RegisterRoutes sets up all the HTTP routes for the server.
func (s *Server) RegisterRoutes() {
	// Register authenticated routes with middleware
	s.Mux.HandleFunc("/", s.AuthMiddleware(handlers.HandlerTxt))
	// Support both legacy and v1-compatible paths
	s.Mux.HandleFunc("/chat/completions", s.AuthMiddleware(handlers.HandlerTxt))
	s.Mux.HandleFunc("/v1/chat/completions", s.AuthMiddleware(handlers.HandlerTxt))
	s.Mux.HandleFunc("/image", s.AuthMiddleware(handlers.HandlerImage))
	s.Mux.HandleFunc("/models", s.AuthMiddleware(handlers.ListModelsHandler))

	// Public endpoints (no auth required)
	s.Mux.HandleFunc("/ping", handlers.PingHandler)
}
