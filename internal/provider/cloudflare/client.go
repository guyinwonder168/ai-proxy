package cloudflare

import (
	"ai-proxy/internal/bufferpool"
	"ai-proxy/internal/client"
	"ai-proxy/internal/errors"
	"ai-proxy/internal/schema"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// pooledReaderReadCloser wraps a bytes.Reader to implement io.ReadCloser
// and returns the reader to the pool when closed
type pooledReaderReadCloser struct {
	reader *bytes.Reader
}

// Throttler handles rate limiting for provider requests
type Throttler struct {
	lastTimeRequest time.Time
	mutex           sync.Mutex
	maxTimeoutTime  time.Duration
}

// NewThrottler creates a new Throttler with the specified timeout
func NewThrottler(maxTimeout time.Duration) *Throttler {
	return &Throttler{
		lastTimeRequest: time.Now(),
		maxTimeoutTime:  maxTimeout,
	}
}

// Wait blocks until the throttling period has passed
// Read implements the io.Reader interface
func (p *pooledReaderReadCloser) Read(b []byte) (int, error) {
	return p.reader.Read(b)
}

// Close implements the io.Closer interface and returns the reader to the pool
func (p *pooledReaderReadCloser) Close() error {
	bufferpool.PutReader(p.reader)
	return nil
}

func (t *Throttler) Wait(modelName string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !time.Now().After(t.lastTimeRequest.Add(t.maxTimeoutTime)) {
		waitTime := time.Until(t.lastTimeRequest.Add(t.maxTimeoutTime))
		log.Printf("Throttled %v sec for %s", waitTime.Seconds(), modelName)
		time.Sleep(waitTime)
	}

	t.lastTimeRequest = time.Now()
}

func CreateRequest(providerURL, model, token string, reqBody schema.RequestOpenAICompatable) (*http.Request, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// fmt.Printf("JSON BODY: %s\n", string(jsonBody))

	// Use a pooled reader instead of bytes.NewReader
	reader := bufferpool.GetReader()
	reader.Reset(jsonBody)

	req, err := http.NewRequest(http.MethodPost, providerURL, reader)
	if err != nil {
		// Return the reader to the pool if there's an error
		bufferpool.PutReader(reader)
		return nil, err
	}

	// Add a function to return the reader to the pool after the request is done
	// We can do this by replacing the request body with a custom ReadCloser
	req.Body = &pooledReaderReadCloser{reader: reader}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	return req, nil
}

func (c *CloudflareClient) callWithClient(reqBody schema.RequestOpenAICompatable) ([]byte, error) {
	// Execute requests with throttling
	c.throttler.Wait(c.Model)

	req, err := CreateRequest(c.ProviderURL, c.Model, c.Token, reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// CloudflareClient implements the Provider interface for Cloudflare
type CloudflareClient struct {
	ProviderURL string
	Model       string
	Token       string
	httpClient  *http.Client
	throttler   *Throttler
}

// NewCloudflareClient creates a new CloudflareClient with the provided configuration
func NewCloudflareClient(providerURL, model, token string, config schema.HttpClientConfig) *CloudflareClient {
	return &CloudflareClient{
		ProviderURL: providerURL,
		Model:       model,
		Token:       token,
		httpClient:  client.NewHttpClient(config),
		throttler:   NewThrottler(1 * time.Second),
	}
}

// Call implements the Provider interface
func (c *CloudflareClient) Call(ctx context.Context, payload []byte, path string) ([]byte, error) {
	var reqBody schema.RequestOpenAICompatable
	err := json.Unmarshal(payload, &reqBody)
	if err != nil {
		return nil, errors.NewValidationError(fmt.Sprintf("error in json.Unmarshal: %v", err))
	}
	return c.callWithClient(reqBody)
}
