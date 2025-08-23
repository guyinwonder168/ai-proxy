package stream

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// RetryableStreamReader wraps an io.ReadCloser to provide retry functionality with SSE keep-alive
type RetryableStreamReader struct {
	originalStream  io.ReadCloser
	streamRecreator func() (io.ReadCloser, error) // Function to recreate the stream
	currentStream   io.ReadCloser                 // Current active stream
	readBuffer      []byte
	bufferPos       int
	bufferLen       int
	isEOF           bool
	retryCount      int
	maxRetries      int
	baseDelay       time.Duration
	maxDelay        time.Duration
	jitter          float64
	onRetry         func(attempt int, err error) // Callback for retry events
}

// NewRetryableStreamReader creates a new RetryableStreamReader
// streamRecreator is optional and can be nil for backward compatibility
func NewRetryableStreamReader(stream io.ReadCloser, streamRecreator func() (io.ReadCloser, error), maxRetries int, baseDelay, maxDelay time.Duration, jitter float64) *RetryableStreamReader {
	return &RetryableStreamReader{
		originalStream:  stream,
		streamRecreator: streamRecreator,
		currentStream:   stream, // Start with the original stream
		readBuffer:      make([]byte, 4096),
		bufferPos:       0,
		bufferLen:       0,
		isEOF:           false,
		retryCount:      0,
		maxRetries:      maxRetries,
		baseDelay:       baseDelay,
		maxDelay:        maxDelay,
		jitter:          jitter,
	}
}

// SetRetryCallback sets a callback function that will be called on each retry attempt
func (r *RetryableStreamReader) SetRetryCallback(fn func(attempt int, err error)) {
	r.onRetry = fn
}

// Read implements the io.Reader interface with retry logic
func (r *RetryableStreamReader) Read(p []byte) (n int, err error) {
	if r.isEOF {
		return 0, io.EOF
	}

	// If we have data in our buffer, return it first
	if r.bufferPos < r.bufferLen {
		n = copy(p, r.readBuffer[r.bufferPos:r.bufferLen])
		r.bufferPos += n
		return n, nil
	}

	// Use the current stream for reading
	currentStream := r.currentStream

	for r.retryCount <= r.maxRetries {
		n, err = currentStream.Read(p)

		// If successful, return the data
		if err == nil || err == io.EOF {
			if err == io.EOF {
				r.isEOF = true
			}
			return n, err
		}

		// If we've reached max retries, return the error
		if r.retryCount >= r.maxRetries {
			return n, fmt.Errorf("max retries exceeded: %w", err)
		}

		// Call the retry callback if set
		if r.onRetry != nil {
			r.onRetry(r.retryCount+1, err)
		}

		// Calculate delay with exponential backoff and jitter
		delay := r.calculateDelay(r.retryCount)

		// Wait for the delay
		time.Sleep(delay)

		// Increment retry count
		r.retryCount++

		// Try to re-establish the stream if we have a recreator function
		if r.streamRecreator != nil {
			// Close the current stream if it's not nil
			if currentStream != nil {
				currentStream.Close()
			}

			// Recreate the stream
			newStream, recreateErr := r.streamRecreator()
			if recreateErr != nil {
				// If we can't recreate the stream, return the original error
				return n, fmt.Errorf("failed to recreate stream: %v, original error: %w", recreateErr, err)
			}

			// Use the new stream for the next attempt
			currentStream = newStream
			r.currentStream = newStream
		}
		// If no streamRecreator is provided, we continue with the same stream
		// This maintains backward compatibility
	}

	return 0, fmt.Errorf("max retries exceeded")
}

// cryptoRandFloat64 generates a cryptographically secure random float64 between 0.0 and 1.0
func cryptoRandFloat64() (float64, error) {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return 0, err
	}

	// Convert bytes to uint64 then to float64 in [0,1)
	u := binary.LittleEndian.Uint64(b[:])
	return float64(u) / (1 << 64), nil
}

// calculateDelay calculates the delay with exponential backoff and jitter
func (r *RetryableStreamReader) calculateDelay(attempt int) time.Duration {
	// Calculate exponential backoff: baseDelay * 2^attempt
	delay := r.baseDelay * time.Duration(1<<uint(attempt))

	// Cap at maxDelay
	if delay > r.maxDelay {
		delay = r.maxDelay
	}

	// Apply jitter: delay * (1 - jitter + random * 2 * jitter)
	random, err := cryptoRandFloat64()
	if err != nil {
		// Fallback to math/rand if crypto/rand fails (should be rare)
		log.Printf("Warning: failed to generate crypto random number, using math/rand: %v", err)
		random = 0.5 // Use a fixed value as fallback
	}
	jitterFactor := 1.0 - r.jitter + random*2*r.jitter
	delay = time.Duration(float64(delay) * jitterFactor)

	return delay
}

// InjectSSEComment injects an SSE comment into the stream
func (r *RetryableStreamReader) InjectSSEComment(comment string) error {
	// Format the comment as an SSE comment
	commentData := fmt.Sprintf(": %s\n\n", comment)

	// Copy existing buffer data to make room for the comment
	if r.bufferLen > r.bufferPos {
		copy(r.readBuffer[r.bufferPos+len(commentData):], r.readBuffer[r.bufferPos:r.bufferLen])
	}

	// Insert the comment at the beginning of the buffer
	copy(r.readBuffer[r.bufferPos:], commentData)

	// Update buffer length
	r.bufferLen += len(commentData)

	return nil
}

// Close implements the io.Closer interface
func (r *RetryableStreamReader) Close() error {
	r.isEOF = true
	// Close the current stream if it's not nil
	if r.currentStream != nil {
		return r.currentStream.Close()
	}
	return nil
}

// GetRetryCount returns the number of retries attempted
func (r *RetryableStreamReader) GetRetryCount() int {
	return r.retryCount
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a network error or timeout
	if err == io.EOF {
		return false // EOF is not retryable as it indicates end of stream
	}

	// Log the error for diagnostic purposes
	log.Printf("IsRetryableError checking error: %v (type: %T)", err, err)

	// For HTTP errors, check status code
	if httpErr, ok := err.(*HTTPError); ok {
		// 5xx errors and 429 (rate limit) are typically retryable
		isRetryable := httpErr.StatusCode >= 500 || httpErr.StatusCode == http.StatusTooManyRequests
		log.Printf("HTTPError - StatusCode: %d, IsRetryable: %t", httpErr.StatusCode, isRetryable)
		return isRetryable
	}

	// Check for OpenRouter errors which have their own retryable logic
	type retryableError interface {
		IsRetryable() bool
	}
	if retryableErr, ok := err.(retryableError); ok {
		isRetryable := retryableErr.IsRetryable()
		log.Printf("RetryableError interface - IsRetryable: %t", isRetryable)
		return isRetryable
	}

	// Check for network errors
	if isNetworkError(err) {
		log.Printf("Network error detected, marking as retryable")
		return true
	}

	// Check for timeout errors
	if isTimeoutError(err) {
		log.Printf("Timeout error detected, marking as retryable")
		return true
	}

	// For other errors, assume they are not retryable by default for safety
	log.Printf("Unknown error type, marking as non-retryable by default")
	return false
}

// isNetworkError checks if an error is a network-related error
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common network error types
	errStr := err.Error()
	networkErrorIndicators := []string{
		"connection refused",
		"connection reset",
		"connection timed out",
		"no such host",
		"network is unreachable",
		"broken pipe",
		"i/o timeout",
	}

	for _, indicator := range networkErrorIndicators {
		if strings.Contains(strings.ToLower(errStr), indicator) {
			return true
		}
	}

	return false
}

// isTimeoutError checks if an error is a timeout-related error
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a context deadline exceeded error
	if err == context.DeadlineExceeded {
		return true
	}

	// Check if it's a timeout error string
	errStr := err.Error()
	timeoutIndicators := []string{
		"timeout",
		"deadline exceeded",
		"i/o timeout",
	}

	for _, indicator := range timeoutIndicators {
		if strings.Contains(strings.ToLower(errStr), indicator) {
			return true
		}
	}

	return false
}

// HTTPError represents an HTTP error with status code
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}
