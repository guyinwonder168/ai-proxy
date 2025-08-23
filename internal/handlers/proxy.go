package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"ai-proxy/internal/bufferpool"
	"ai-proxy/internal/config"
	"ai-proxy/internal/errors"
	"ai-proxy/internal/provider/cloudflare"
	"ai-proxy/internal/provider/gemini"
	"ai-proxy/internal/provider/gigachat"
	"ai-proxy/internal/provider/groq"
	"ai-proxy/internal/provider/openai"
	"ai-proxy/internal/provider/openrouter"
	"ai-proxy/internal/schema"
	"ai-proxy/internal/stream"

	"github.com/tidwall/gjson"
)

// pooledReadCloser wraps a bytes.Reader to implement io.ReadCloser
// and returns the reader to the pool when closed
type pooledReadCloser struct {
	reader *bytes.Reader
}

var (
	// Retry configuration
	retryMaxRetries int
	retryBaseDelay  time.Duration
	retryMaxDelay   time.Duration
	retryJitter     float64
)

// Config represents the configuration structure for the proxy
type Config struct {
	Models []config.Model `yaml:"models"`
}

// SetRetryConfig sets the retry configuration for streaming requests
func SetRetryConfig(maxRetries int, baseDelay, maxDelay time.Duration, jitter float64) {
	retryMaxRetries = maxRetries
	retryBaseDelay = baseDelay
	retryMaxDelay = maxDelay
	retryJitter = jitter
}

func HandlerTxt(w http.ResponseWriter, req *http.Request) {

	var (
		modelName, modelSize string
		response             []byte
		err                  error
	)

	if req.Method != http.MethodPost {
		http.Error(w, "", http.StatusServiceUnavailable)

		return
	}

	reqBodyBytes, err := io.ReadAll(req.Body)
	//log.Printf("Request-body: %s \n", string(reqBodyBytes))
	if err != nil {
		proxyErr := errors.NewValidationError(fmt.Sprintf("Failed to read request body: %v", err))
		http.Error(w, proxyErr.Error(), proxyErr.Code)
		return
	}

	if !gjson.ValidBytes(reqBodyBytes) {
		proxyErr := errors.NewValidationError("Invalid request body")
		http.Error(w, proxyErr.Error(), proxyErr.Code)
		return
	}

	modelName = gjson.GetBytes(reqBodyBytes, "model").String()

	if len(modelName) < 10 {
		if modelName != "BIG" {
			modelSize = "SMALL"
		} else {
			modelSize = "BIG"
		}

		for i := 0; i < 5; i++ {
			modelName = selectModel(modelSize, len(reqBodyBytes))
			if modelName == "" {
				log.Printf("No available models for this request length = %d", len(reqBodyBytes))
				proxyErr := errors.NewModelSelectionError("No available models for this request length", nil)
				http.Error(w, proxyErr.Error(), proxyErr.Code)
				return
			}

			stream, isStream, err := sendRequestToLLM(modelName, reqBodyBytes, req.URL.Path)
			if err != nil {
				setMaxLimitMinute(modelName) // set max minuteCount for pause after error
				log.Printf("Error sending request to LLM: %v", err)

				// Check if it's a rate limit error
				if errors.IsAIProxyError(err) {
					if proxyErr, ok := err.(*errors.AIProxyError); ok && proxyErr.IsRateLimitError() {
						// Continue to try another model
						continue
					}
				}

				// For other errors, return immediately
				return
			}

			// For streaming responses, we need to handle them differently
			if isStream {
				// Set the Content-Type header for streaming
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")

				// Stream the response directly to the client
				_, err = io.Copy(w, stream)
				if err != nil {
					log.Printf("Error streaming response: %v", err)
				}

				// Close the stream
				stream.Close()

				return
			} else {
				// For non-streaming responses, read the response into a byte array
				response, err = io.ReadAll(stream)
				if err != nil {
					log.Printf("Error reading response: %v", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				// Close the stream
				stream.Close()
			}

			break
		}
	} else {
		stream, isStream, err := sendRequestToLLM(modelName, reqBodyBytes, req.URL.Path)
		if err != nil {
			// Check if it's already an AIProxyError
			if errors.IsAIProxyError(err) {
				if proxyErr, ok := err.(*errors.AIProxyError); ok {
					http.Error(w, proxyErr.Error(), proxyErr.Code)
					return
				}
			}

			// Default to internal server error
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// For streaming responses, we need to handle them differently
		if isStream {
			// Set the Content-Type header for streaming
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			// Stream the response directly to the client
			log.Printf("sending response-Client")
			_, err = io.Copy(w, stream)
			if err != nil {
				log.Printf("Error streaming response: %v", err)
			}

			// Close the stream
			stream.Close()

			return
		} else {
			// For non-streaming responses, read the response into a byte array
			response, err = io.ReadAll(stream)
			if err != nil {
				log.Printf("Error reading response: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Close the stream
			stream.Close()
		}
	}
	// Sending response to client (content not logged for security)
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func setMaxLimitMinute(modelName string) {
	var model config.Model

	for _, m := range config.Models {
		if m.Name == modelName {
			model = m
			break
		}
	}

	limit := config.RateLimits[modelName]

	limit.Lock()
	defer limit.Unlock()

	limit.SetCounts(model.RequestsPerMin+1, limit.GetHourCount(), limit.GetDayCount())
	_, lastHour, lastDay := limit.GetLastTimes()
	limit.SetLastTimes(time.Now(), lastHour, lastDay)
}

func selectModel(modelSize string, requestLength int) string {
	var selectedModel *config.Model
	var selectedLastRequest time.Time

	for _, model := range config.Models {
		if model.Size != modelSize {
			continue
		}

		if requestLength > model.MaxRequestLength {
			continue
		}

		limit := config.RateLimits[model.Name]
		limit.Lock()

		// Check if model is within rate limits
		withinLimits := limit.GetMinuteCount() < model.RequestsPerMin &&
			limit.GetHourCount() < model.RequestsPerHour &&
			limit.GetDayCount() < model.RequestsPerDay

		if withinLimits {
			// Select the model with the lowest priority
			// If priorities are equal, select the one with the earliest lastRequest
			shouldSelect := false
			if selectedModel == nil {
				shouldSelect = true
			} else if model.Priority < selectedModel.Priority {
				shouldSelect = true
			} else if model.Priority == selectedModel.Priority {
				if limit.GetLastRequest().Before(time.Now().Add(-time.Hour)) && selectedLastRequest.Before(time.Now().Add(-time.Hour)) &&
					model.MaxRequestLength < selectedModel.MaxRequestLength {
					shouldSelect = true
				} else if limit.GetLastRequest().Before(selectedLastRequest) {
					shouldSelect = true
				}
			}

			if shouldSelect {
				selectedModel = &model
				selectedLastRequest = limit.GetLastRequest()
			}
		}

		limit.Unlock()
	}

	// If we found a model, reserve it by incrementing its rate limits
	if selectedModel != nil {
		limit := config.RateLimits[selectedModel.Name]
		limit.Lock()

		// Double-check that the model is still within rate limits
		stillWithinLimits := limit.GetMinuteCount() < selectedModel.RequestsPerMin &&
			limit.GetHourCount() < selectedModel.RequestsPerHour &&
			limit.GetDayCount() < selectedModel.RequestsPerDay

		if stillWithinLimits {
			// Increment its rate limits to reserve it
			now := time.Now()
			updateLimitCounters(limit, now)
			limit.SetCounts(limit.GetMinuteCount()+1, limit.GetHourCount()+1, limit.GetDayCount()+1)
			limit.SetLastRequest(now)
			limit.Unlock()
			return selectedModel.Name
		}

		limit.Unlock()
	}

	return ""
}

func updateLimitCounters(limit *config.RateLimit, now time.Time) {
	lastMinute, lastHour, lastDay := limit.GetLastTimes()

	// Initialize times if they're zero values
	if lastMinute.IsZero() {
		lastMinute = now
	}
	if lastHour.IsZero() {
		lastHour = now
	}
	if lastDay.IsZero() {
		lastDay = now
	}

	// Check if we need to reset minute counter
	if now.Sub(lastMinute) >= time.Minute {
		limit.SetCounts(0, limit.GetHourCount(), limit.GetDayCount())
		lastMinute = now
	}

	// Check if we need to reset hour counter
	if now.Sub(lastHour) >= time.Hour {
		limit.SetCounts(limit.GetMinuteCount(), 0, limit.GetDayCount())
		lastHour = now
	}

	// Check if we need to reset day counter
	if now.Sub(lastDay) >= 24*time.Hour {
		limit.SetCounts(limit.GetMinuteCount(), limit.GetHourCount(), 0)
		lastDay = now
	}

	limit.SetLastTimes(lastMinute, lastHour, lastDay)
}

// not used at the moment
// func incrementRateLimit(modelName string) {
// 	limit := config.RateLimits[modelName]

// 	limit.mux.Lock()
// 	defer limit.mux.Unlock()

// 	now := time.Now()

// 	updateLimitCounters(limit, now)

// 	limit.minuteCount++
// 	limit.hourCount++
// 	limit.dayCount++
// 	limit.lastRequest = now
// }

func getModelByName(modelName string) (config.Model, bool) {
	for _, model := range config.Models {
		if model.Name == modelName {
			// Rate limits are already incremented in selectModel, so we don't need to do it here
			return model, true
		}
	}

	return config.Model{}, false
}

//not used at the moment
// func getRequestLength(requestBody schema.RequestOpenAICompatable) int {
// 	var res int

// 	for _, v := range requestBody.Messages {
// 		res += len(v.Content)
// 	}

// 	return res
// }

// func sendRequestToLLM(modelName string, requestBody schema.RequestOpenAICompatable) ([]byte, error) {
func sendRequestToLLM(modelName string, requestBody []byte, reqPath string) (io.ReadCloser, bool, error) {
	var resp []byte

	var err error

	log.Printf("Request to model: %s\n", modelName)

	model, found := getModelByName(modelName)
	if !found {
		return nil, false, errors.NewConfigurationError(fmt.Sprintf("specified model not found - %s", modelName), nil)
	}

	switch model.Provider {
	case "cloudflare":
		log.Printf("Protocol: Cloudflare")
		provider := cloudflare.NewCloudflareClient(model.URL, "@"+model.Name, model.Token, model.HttpClientConfig)
		resp, err = provider.Call(context.Background(), requestBody, reqPath)
	case "google": // todo change on openai.Call - https://developers.googleblog.com/en/gemini-is-now-accessible-from-the-openai-library/
		log.Printf("Protocol: Google")
		provider := gemini.NewGeminiClient(model.URL, model.Name, model.Token, model.HttpClientConfig)
		resp, err = provider.Call(context.Background(), requestBody, reqPath)
	case "gigachat":
		log.Printf("Protocol: Cohere")
		provider := &gigachat.GigaChatClient{
			ProviderURL: model.URL,
			Model:       strings.TrimPrefix(model.Name, model.Provider+"/"),
			Token:       model.Token,
		}
		resp, err = provider.Call(context.Background(), requestBody, reqPath)
	case "groq", "arliai", "github":
		log.Printf("Protocol: groq, arliai, github")
		provider := groq.NewGroqClient(model.URL, strings.TrimPrefix(model.Name, model.Provider+"/"), model.Token, model.HttpClientConfig)
		resp, err = provider.Call(context.Background(), requestBody, reqPath)
	case "cohere":
		log.Printf("Protocol: Cohere")
		provider := openai.NewOpenAIClient(model.URL, strings.TrimPrefix(model.Name, model.Provider+"/"), model.Token, model.HttpClientConfig)
		resp, err = provider.Call(context.Background(), requestBody, reqPath)
		if err == nil {
			response := schema.ResponseOpenAICompatable{
				Model: model.Name,
				Choices: []struct {
					Index   int `json:"index,omitempty"`
					Message struct {
						Role    string `json:"role,omitempty"`
						Content string `json:"content,omitempty"`
					} `json:"message,omitempty"`
					FinishReason string `json:"finish_reason,omitempty"`
				}{
					{
						Index: 0,
						Message: struct {
							Role    string `json:"role,omitempty"`
							Content string `json:"content,omitempty"`
						}{
							Role:    "assistant",
							Content: gjson.GetBytes(resp, "message.content.0.text").String(),
						},
						FinishReason: "stop",
					},
				},
			}
			resp, err = json.Marshal(response)
		}
	case "openrouter":
		log.Printf("Protocol: OpenRouter")
		provider := openrouter.NewOpenRouterClient(model.URL, strings.TrimPrefix(model.Name, model.Provider+"/"), model.Token, model.HttpClientConfig)
		// Check if the request is a streaming request
		isStream := gjson.GetBytes(requestBody, "stream").Bool()
		if isStream {
			// For streaming requests, call StreamCall and wrap with retryable stream
			streamResponse, err := provider.StreamCall(requestBody)
			if err != nil {
				return nil, false, err
			}

			// Create a stream recreator function that captures the necessary parameters
			streamRecreator := func() (io.ReadCloser, error) {
				return provider.StreamCall(requestBody)
			}

			// Wrap the stream with retry functionality
			retryableStream := stream.NewRetryableStreamReader(streamResponse, streamRecreator, retryMaxRetries, retryBaseDelay, retryMaxDelay, retryJitter)

			// Set up retry callback for metrics collection
			retryableStream.SetRetryCallback(func(attempt int, retryErr error) {
				log.Printf("Retry attempt %d for model %s: %v", attempt, modelName, retryErr)
				// Here you could add metrics collection
			})

			return retryableStream, true, nil
		} else {
			resp, err = provider.Call(context.Background(), requestBody, reqPath)
		}
	default:
		log.Printf("Protocol: OpenAI")
		provider := openai.NewOpenAIClient(model.URL, strings.TrimPrefix(model.Name, model.Provider+"/"), model.Token, model.HttpClientConfig)
		resp, err = provider.Call(context.Background(), requestBody, reqPath)
	}

	if err != nil {
		log.Printf("ERROR from provider %s: %v\n", modelName, err)

		// Check if it's already an AIProxyError
		if errors.IsAIProxyError(err) {
			return nil, false, err
		}

		// Check if it's an OpenRouterError that can be converted
		if openRouterErr, ok := err.(*schema.OpenRouterError); ok {
			return nil, false, openRouterErr.ToAIProxyError(modelName)
		}

		// Default to a provider error
		return nil, false, errors.NewProviderError(fmt.Sprintf("Error from provider %s: %v", modelName, err), model.Provider, modelName, 500, err)
	}

	if resp == nil {
		return nil, false, errors.NewProviderError("no response from LLM", model.Provider, modelName, 500, nil)
	}
	// Trim leading and trailing whitespace from the entire response body
	resp = []byte(strings.TrimSpace(string(resp)))
	content := gjson.GetBytes(resp, "choices.0.message.content").String()

	if len(content) == 0 {
		log.Printf("ERROR: Empty content detected, returning early")
		return nil, false, errors.NewProviderError("no content", model.Provider, modelName, 500, nil)
	}
	// as an proxy why we have to filter the <think> part ? shouldnt it client that has to take care?

	// // replace block <think>...</think> in reasoning models
	// index := strings.Index(content, "</think>")
	// if index != -1 {
	// 	content = content[index+len("</think>"):]
	//
	// 	resp, err = sjson.SetBytes(resp, "choices.0.message.content", strings.TrimSpace(content))
	// 	if err != nil {
	// 		return nil, false, fmt.Errorf("error sjson.SetBytes in replace think: %w", err)
	// 	}
	// }

	// Convert the response to an io.ReadCloser for non-streaming responses
	reader := bufferpool.GetReader()
	reader.Reset(resp)
	return &pooledReadCloser{reader: reader}, false, nil
}

//not used at the moment
// func printFirstChars(data string) string {
// 	if len(data) > 100 {
// 		return strings.TrimSpace(data[:100])
// 	}

// 	return data
// }

// Read implements the io.Reader interface
func (p *pooledReadCloser) Read(b []byte) (int, error) {
	return p.reader.Read(b)
}

// Close implements the io.Closer interface and returns the reader to the pool
func (p *pooledReadCloser) Close() error {
	bufferpool.PutReader(p.reader)
	return nil
}
