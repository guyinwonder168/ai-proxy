package handlers

import (
	"bytes"
	"cmp"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"ai-proxy/internal/bufferpool"
	"ai-proxy/internal/config"

	"github.com/tidwall/gjson"
)

// pooledBufferReadCloser wraps a byte slice to implement io.ReadCloser
// and returns the buffer to the pool when closed
type pooledBufferReadCloser struct {
	buf []byte
	pos int
}

// Example request
// curl https://ai-proxy-evgensoft.koyeb.app/image -d '{"model": "huggingface/black-forest-labs/FLUX.1-dev", "prompt": "Cat sleep on the moon"}'

type RequestGenerateImage struct {
	Model  string `json:"model,omitempty"`
	Prompt string `json:"prompt,omitempty"`
	Inputs string `json:"inputs,omitempty"`
}

type TogetherRequestGenerateImage struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	ResponseFormat string `json:"response_format"`
}

func HandlerImage(w http.ResponseWriter, req *http.Request) {
	var (
		requestBody RequestGenerateImage
		response    []byte
		err         error
	)

	err = json.NewDecoder(req.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	prompt := cmp.Or(requestBody.Prompt, requestBody.Inputs)

	if len(prompt) == 0 {
		http.Error(w, "Empty prompt", http.StatusBadRequest)
		return
	}

	if requestBody.Model == "" || requestBody.Model == "all" {
		for _, model := range config.Models {
			if model.MaxRequestLength != 0 {
				continue
			}

			// Check rate limits with proper locking
			limit := config.RateLimits[model.Name]
			limit.Lock()
			overLimit := limit.MinuteCount >= model.RequestsPerMin ||
				limit.HourCount >= model.RequestsPerHour ||
				limit.DayCount >= model.RequestsPerDay
			limit.Unlock()

			if overLimit {
				continue
			}

			response, err = RequestProvider(model.Name, prompt)
			if err != nil {
				setMaxLimitMinute(model.Name) // set max minuteCount to pause after an error
				log.Printf("ERROR: %s, body length: %d\n", err, len(response))
				continue
			}

			break
		}
	} else {
		response, err = RequestProvider(requestBody.Model, prompt)
	}

	if err != nil {
		log.Printf("ERROR: %s, body length: %d\n", err, len(response))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Get image in %d bytes\n", len(response))

	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(response)
}

func RequestProvider(modelName, prompt string) ([]byte, error) {
	model, found := getModelByName(modelName)
	if !found {
		return nil, fmt.Errorf("specified model not found - %s", modelName)
	}

	log.Printf("Request to image model: %s\n", modelName)

	if model.Provider == "airforce" {
		return getAairforceImagine(model.URL, prompt, strings.TrimPrefix(model.Name, model.Provider+"/"))
	}

	data, err := generatePayload(model, prompt)
	if err != nil {
		return nil, err
	}

	// Use a pooled buffer instead of bytes.NewBuffer
	buf := bufferpool.GetBuffer(len(data))
	buf = append(buf, data...)
	req, err := http.NewRequest(http.MethodPost, model.URL, bytes.NewReader(buf))
	if err != nil {
		// Return the buffer to the pool if there's an error
		bufferpool.PutBuffer(buf)
		return nil, err
	}

	// Add a function to return the buffer to the pool after the request is done
	// We can do this by replacing the request body with a custom ReadCloser
	req.Body = &pooledBufferReadCloser{buf: buf}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", model.Token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if len(body) < 500 {
		return body, fmt.Errorf("small response length: %d", len(body))
	}

	switch model.Name {
	case "cloudflare/black-forest-labs/flux-1-schnell":
		return base64.StdEncoding.DecodeString(gjson.GetBytes(body, "result.image").String())
	case "together/black-forest-labs/FLUX.1-schnell-Free", "aimlapi/flux/schnell":
		return base64.StdEncoding.DecodeString(gjson.GetBytes(body, "data.0.b64_json").String())
	default:
		return body, nil
	}
}

func generatePayload(model config.Model, prompt string) ([]byte, error) {
	var data []byte
	var err error

	switch model.Provider {
	case "huggingface":
		var payload RequestGenerateImage
		payload.Inputs = prompt
		data, err = json.Marshal(payload)

	// case "cloudflare":

	case "together", "aimlapi":
		var payload TogetherRequestGenerateImage
		payload.Model = strings.TrimPrefix(model.Name, model.Provider+"/")
		payload.Prompt = prompt
		payload.ResponseFormat = "b64_json"
		data, err = json.Marshal(payload)

	default:
		var payload RequestGenerateImage
		payload.Prompt = prompt
		data, err = json.Marshal(payload)
	}

	return data, err
}

// Read implements the io.Reader interface
func (p *pooledBufferReadCloser) Read(b []byte) (int, error) {
	if p.pos >= len(p.buf) {
		return 0, io.EOF
	}

	n := copy(b, p.buf[p.pos:])
	p.pos += n
	return n, nil
}

// Close implements the io.Closer interface and returns the buffer to the pool
func (p *pooledBufferReadCloser) Close() error {
	bufferpool.PutBuffer(p.buf)
	p.buf = nil
	p.pos = 0
	return nil
}

func getAairforceImagine(baseURL, prompt, model string) ([]byte, error) {
	// Build URL with query parameters
	params := url.Values{}
	params.Add("prompt", prompt)
	params.Add("model", model)

	// Construct the full URL
	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Execute the GET request
	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if len(body) < 500 {
		return body, fmt.Errorf("small response length: %d", len(body))
	}

	return body, nil
}
