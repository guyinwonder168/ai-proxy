package openrouter

import (
	"bytes"
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// transformResponse standardizes the OpenRouter response based on the request path
func transformResponse(body []byte, path string) ([]byte, error) {
	// Trim leading and trailing whitespace from the response body
	body = bytes.TrimSpace(body)

	switch path {
	case "/v1/chat/completions":
		// Check if the response has the correct object field
		object := gjson.GetBytes(body, "object").String()
		if object != "chat.completion" {
			// Set the object field to "chat.completion"
			modifiedBody, err := sjson.SetBytes(body, "object", "chat.completion")
			if err != nil {
				return nil, fmt.Errorf("error setting object field: %w", err)
			}
			return modifiedBody, nil
		}
		return body, nil
	default:
		// For all other paths, return the body unmodified
		return body, nil
	}
}
