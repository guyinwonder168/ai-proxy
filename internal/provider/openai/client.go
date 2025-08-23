package openai

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

	"github.com/tidwall/sjson"
)

// pooledReaderReadCloser wraps a bytes.Reader to implement io.ReadCloser
// and returns the reader to the pool when closed
type pooledReaderReadCloser struct {
	reader *bytes.Reader
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

func (c *OpenAIClient) callWithClient(requestBody []byte) ([]byte, error) {
	reqBody, err := sjson.SetBytes(requestBody, "model", c.Model)
	if err != nil {
		return nil, errors.NewValidationError(fmt.Sprintf("error in sjson.SetBytes: %v", err))
	}

	// Use a pooled reader instead of bytes.NewReader
	reader := bufferpool.GetReader()
	reader.Reset(reqBody)

	req, err := http.NewRequest(http.MethodPost, c.ProviderURL, reader)
	if err != nil {
		// Return the reader to the pool if there's an error
		bufferpool.PutReader(reader)
		return nil, err
	}

	// Add a function to return the reader to the pool after the request is done
	// We can do this by replacing the request body with a custom ReadCloser
	req.Body = &pooledReaderReadCloser{reader: reader}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return body, errors.NewProviderError(fmt.Sprintf("unexpected status code: %d", resp.StatusCode), "OpenAI", c.Model, resp.StatusCode, nil)
	}

	return body, nil
}

// OpenAIClient implements the Provider interface for OpenAI
type OpenAIClient struct {
	ProviderURL string
	Model       string
	Token       string
	httpClient  *http.Client
}

// NewOpenAIClient creates a new OpenAIClient with the provided configuration
func NewOpenAIClient(providerURL, model, token string, config schema.HttpClientConfig) *OpenAIClient {
	return &OpenAIClient{
		ProviderURL: providerURL,
		Model:       model,
		Token:       token,
		httpClient:  client.NewHttpClient(config),
	}
}

// Call implements the Provider interface
func (c *OpenAIClient) Call(ctx context.Context, payload []byte, path string) ([]byte, error) {
	return c.callWithClient(payload)
}
