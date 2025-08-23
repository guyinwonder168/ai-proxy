// Package handlers provides HTTP handlers for the AI proxy service.
package handlers

import (
	"net/http"
)

// PingHandler handles the ping endpoint for health checks.
func PingHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("OK"))
}
