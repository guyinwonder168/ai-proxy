// Package handlers provides HTTP handlers for the AI proxy service.
package handlers

import (
	"ai-proxy/internal/config"
	"encoding/json"
	"net/http"
)

// ListModelsHandler handles the /models endpoint to list available models.
func ListModelsHandler(w http.ResponseWriter, req *http.Request) {
	type Data struct {
		ID string `json:"id"`
	}

	type Models struct {
		Object string `json:"object"`
		Data   []Data `json:"data"`
	}

	var models Models

	models.Object = "list"

	for _, v := range config.Models {
		models.Data = append(models.Data, Data{ID: v.Name})
	}

	models.Data = append(models.Data, Data{ID: "SMALL"})
	models.Data = append(models.Data, Data{ID: "BIG"})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}
