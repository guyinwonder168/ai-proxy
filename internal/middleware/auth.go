// Package middleware provides HTTP middleware functions for the AI proxy service.
package middleware

import (
	"net/http"

	"ai-proxy/internal/errors"
)

// AuthMiddleware creates a middleware function that validates the Authorization header.
func AuthMiddleware(authToken string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			if authToken == "" {
				proxyErr := errors.NewConfigurationError("Authentication configuration error", nil)
				http.Error(w, proxyErr.Error(), proxyErr.Code)
				return
			}
			if req.Header.Get("Authorization") != "Bearer "+authToken {
				// Set proper WWW-Authenticate header
				w.Header().Set("WWW-Authenticate", `Bearer realm="AI Proxy"`)
				// Return generic "Unauthorized" message without details
				proxyErr := errors.NewAuthenticationError("Unauthorized")
				http.Error(w, proxyErr.Error(), proxyErr.Code)
				// Log failed attempts without sensitive data
				// log.Printf("Unauthorized access attempt from %s", req.RemoteAddr)
				return
			}
			// If authorized, call the next handler
			next(w, req)
		}
	}
}
