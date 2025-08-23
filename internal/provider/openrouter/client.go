package openrouter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"ai-proxy/internal/bufferpool"
	"ai-proxy/internal/client"
	"ai-proxy/internal/errors"
	"ai-proxy/internal/schema"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// pooledReaderReadCloser wraps a bytes.Reader to implement io.ReadCloser
// and returns the reader to the pool when closed
type pooledReaderReadCloser struct {
	reader *bytes.Reader
}

// Call sends a request to the OpenRouter API using the provided HTTP client
func Call(providerURL, model, token string, requestBody []byte, httpClient *http.Client) ([]byte, error) {
	// Set the model in the request body
	reqBody, err := sjson.SetBytes(requestBody, "model", model)
	if err != nil {
		return nil, fmt.Errorf("error in sjson.SetBytes: %w", err)
	}

	// Use a pooled reader instead of bytes.NewReader
	reader := bufferpool.GetReader()
	reader.Reset(reqBody)

	req, err := http.NewRequest(http.MethodPost, providerURL, reader)
	if err != nil {
		// Return the reader to the pool if there's an error
		bufferpool.PutReader(reader)
		return nil, err
	}

	// Add a function to return the reader to the pool after the request is done
	// We can do this by replacing the request body with a custom ReadCloser
	req.Body = &pooledReaderReadCloser{reader: reader}

	// Set required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Set OpenRouter-specific headers
	req.Header.Set("X-API-Source", "ai-proxy")
	req.Header.Set("HTTP-Referer", "https://github.com/edwardhook/ai-proxy")

	// Send the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		// Parse the error response
		openRouterErr := schema.ParseOpenRouterError(resp.StatusCode, body)
		return body, openRouterErr
	}
	//log.Printf("Body: , %s\n", string(body))
	return body, nil
}

// Read implements the io.Reader interface
func (p *pooledReaderReadCloser) Read(b []byte) (n int, err error) {
	return p.reader.Read(b)
}

// Close implements the io.Closer interface and returns the reader to the pool
func (p *pooledReaderReadCloser) Close() error {
	bufferpool.PutReader(p.reader)
	return nil
}

// StreamCall sends a streaming request to the OpenRouter API using the provided HTTP client
func StreamCall(providerURL, model, token string, requestBody []byte, httpClient *http.Client) (io.ReadCloser, error) {
	// Set the model in the request body
	reqBody, err := sjson.SetBytes(requestBody, "model", model)
	if err != nil {
		return nil, fmt.Errorf("error in sjson.SetBytes: %w", err)
	}

	// Use a pooled reader instead of bytes.NewReader
	reader := bufferpool.GetReader()
	reader.Reset(reqBody)

	req, err := http.NewRequest(http.MethodPost, providerURL, reader)
	if err != nil {
		// Return the reader to the pool if there's an error
		bufferpool.PutReader(reader)
		return nil, err
	}

	// Add a function to return the reader to the pool after the request is done
	// We can do this by replacing the request body with a custom ReadCloser
	req.Body = &pooledReaderReadCloser{reader: reader}

	// Set required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Set OpenRouter-specific headers
	req.Header.Set("X-API-Source", "ai-proxy")
	req.Header.Set("HTTP-Referer", "https://github.com/edwardhook/ai-proxy")

	// Send the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		// Read the response body to include in the error
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			resp.Body.Close()
			return nil, errors.NewProviderError(fmt.Sprintf("unexpected status code: %d", resp.StatusCode), "OpenRouter", "", resp.StatusCode, readErr)
		}
		resp.Body.Close()

		// Parse the error response
		openRouterErr := schema.ParseOpenRouterError(resp.StatusCode, body)
		return nil, openRouterErr
	}

	// Return the response body for streaming
	return resp.Body, nil
}

// OpenRouterClient implements the Provider interface for OpenRouter
type OpenRouterClient struct {
	ProviderURL string
	Model       string
	Token       string
	httpClient  *http.Client
}

// NewOpenRouterClient creates a new OpenRouterClient with the provided configuration
func NewOpenRouterClient(providerURL, model, token string, config schema.HttpClientConfig) *OpenRouterClient {
	return &OpenRouterClient{
		ProviderURL: providerURL,
		Model:       model,
		Token:       token,
		httpClient:  client.NewHttpClient(config),
	}
}

// Call implements the Provider interface
func (c *OpenRouterClient) Call(ctx context.Context, payload []byte, path string) ([]byte, error) {
	// Check if the request is a streaming request
	isStream := gjson.GetBytes(payload, "stream").Bool()

	if isStream {
		// For streaming requests, we return an error indicating that streaming should be handled at a higher level
		return nil, errors.NewConfigurationError("streaming requests should be handled at a higher level", nil)
	}

	resp, err := Call(c.ProviderURL, c.Model, c.Token, payload, c.httpClient)
	if err != nil {
		return nil, err
	}

	return transformResponse(resp, path)
}

// StreamCall sends a streaming request to the OpenRouter API using the client's HTTP client
func (c *OpenRouterClient) StreamCall(requestBody []byte) (io.ReadCloser, error) {
	return StreamCall(c.ProviderURL, c.Model, c.Token, requestBody, c.httpClient)
}
