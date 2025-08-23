package gigachat

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// --- Structures for parsing GigaChat response ---

type GigaChatResponse struct {
	Choices []GigaChatChoice `json:"choices"`
	Created int64            `json:"created"` // Unix timestamp in seconds
	Model   string           `json:"model"`
	Usage   GigaChatUsage    `json:"usage"`
	Object  string           `json:"object"` // "completion" or "chat.completion"
}

type GigaChatChoice struct {
	Message      GigaChatMessage `json:"message"`
	Index        int             `json:"index"`
	FinishReason string          `json:"finish_reason"`
}

type GigaChatMessage struct {
	Content      string                `json:"content"` // Can be an empty string
	Role         string                `json:"role"`
	FunctionCall *GigaChatFunctionCall `json:"function_call,omitempty"`
	// Ignore functions_state_id
}

type GigaChatFunctionCall struct {
	Name      string      `json:"name"`
	Arguments interface{} `json:"arguments"` // Arguments can be a complex JSON object/array
}

type GigaChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	SystemTokens     int `json:"system_tokens,omitempty"` // Added according to the GigaChat API
}

// --- Structures for forming response in OpenAI format ---

type OpenAIResponse struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"` // Unix timestamp in seconds
	Model             string         `json:"model"`
	Choices           []OpenAIChoice `json:"choices"`
	Usage             OpenAIUsage    `json:"usage"` // Not a pointer, because GigaChat docs assume it is present
	SystemFingerprint string         `json:"system_fingerprint"`
}

type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	Logprobs     interface{}   `json:"logprobs"`      // Always null in the Python example
	FinishReason string        `json:"finish_reason"` // Copied from GigaChat
}

type OpenAIMessage struct {
	Role         string              `json:"role"`
	Content      *string             `json:"content"`           // Pointer to allow null; not omitempty so null is serialized
	Refusal      interface{}         `json:"refusal,omitempty"` // Always null in the Python example; uses omitempty
	FunctionCall *OpenAIFunctionCall `json:"function_call,omitempty"`
	ToolCalls    []OpenAIToolCall    `json:"tool_calls,omitempty"`
}

type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // Arguments must be a JSON string
}

type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"` // "function"
	Function OpenAIFunctionCall `json:"function"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	SystemTokens     int `json:"system_tokens,omitempty"` // Added to match the Python proxy
}

// ConvertGigaChatResponseToOpenAI converts a GigaChat response to the OpenAI Chat Completion format.
// gigaChatResponseBytes: raw JSON response from the GigaChat API.
// openAIModelName: the model name to place in the 'model' field of the OpenAI response.
// isToolCall: if true, convert function_call to tool_calls.
func ConvertGigaChatResponseToOpenAI(gigaChatResponseBytes []byte, openAIModelName string, isToolCall bool) ([]byte, error) {
	var gigaResp GigaChatResponse

	err := json.Unmarshal(gigaChatResponseBytes, &gigaResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal GigaChat response: %w", err)
	}

	// Generate UUID for system_fingerprint
	u := uuid.New()
	uuidBytes := u[:]
	shortHexID := hex.EncodeToString(uuidBytes)[:8]

	openAIResp := OpenAIResponse{
		ID:                fmt.Sprintf("chatcmpl-%s", uuid.NewString()),
		Object:            "chat.completion", // Non-streaming response, as in the Python version
		Created:           time.Now().Unix(), // Unix timestamp in seconds
		Model:             openAIModelName,
		SystemFingerprint: fmt.Sprintf("fp_%s", shortHexID),
		Usage: OpenAIUsage{ // Set directly because we expect GigaChat to always return usage
			PromptTokens:     gigaResp.Usage.PromptTokens,
			CompletionTokens: gigaResp.Usage.CompletionTokens,
			TotalTokens:      gigaResp.Usage.TotalTokens,
			SystemTokens:     gigaResp.Usage.SystemTokens, // Included to mirror the Python code
		},
		Choices: make([]OpenAIChoice, 0, len(gigaResp.Choices)),
	}

	for _, gigaChoice := range gigaResp.Choices {
		openAIChoice := OpenAIChoice{
			Index:        0,   // Hardcoded to match the Python code
			Logprobs:     nil, // Hardcoded to match the Python code
			FinishReason: gigaChoice.FinishReason,
			Message: OpenAIMessage{
				Role:    gigaChoice.Message.Role,
				Refusal: nil, // Hardcoded to match the Python code; with omitempty it will be absent
			},
		}

		// Content handling: set to nil if it's empty AND there is a function call
		// Python:
		// if choice["message"].get("content") == "" and choice["message"].get("function_call"):
		//     choice["message"]["content"] = None
		// In Go: Content will be "" if GigaChat sent "" or omitted the field (for string type)
		if gigaChoice.Message.Content == "" && gigaChoice.Message.FunctionCall != nil {
			openAIChoice.Message.Content = nil
		} else {
			// Copy the value to obtain a pointer.
			// If Content is an empty string and there is no FunctionCall, keep "" (not null).
			content := gigaChoice.Message.Content
			openAIChoice.Message.Content = &content
		}

		// Handle FunctionCall / ToolCalls
		// Python: if choice["message"]["role"] == "assistant" and choice["message"].get("function_call"):
		if gigaChoice.Message.Role == "assistant" && gigaChoice.Message.FunctionCall != nil {
			// Serialize arguments to a JSON string
			// Python: arguments = json.dumps(..., ensure_ascii=False)
			// Go: json.Marshal preserves UTF-8 by default (similar to ensure_ascii=False)
			argsBytes, err := json.Marshal(gigaChoice.Message.FunctionCall.Arguments)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal function call arguments for choice %d: %w", gigaChoice.Index, err)
			}

			argsString := string(argsBytes)

			oaFuncCall := OpenAIFunctionCall{
				Name:      gigaChoice.Message.FunctionCall.Name,
				Arguments: argsString,
			}

			if isToolCall {
				// Python:
				// choice["message"]["tool_calls"] = [{"id": ..., "type": "function",
				//   "function": choice["message"].pop("function_call")}]
				toolCall := OpenAIToolCall{
					ID:       fmt.Sprintf("call_%s", uuid.NewString()), // Generate an ID for the tool_call
					Type:     "function",
					Function: oaFuncCall,
				}
				openAIChoice.Message.ToolCalls = append(openAIChoice.Message.ToolCalls, toolCall)
				openAIChoice.Message.FunctionCall = nil // Do not set explicitly; omitempty will drop it
			} else {
				// Python: choice["message"].pop("tool_calls", None)
				openAIChoice.Message.FunctionCall = &oaFuncCall
				openAIChoice.Message.ToolCalls = nil // Do not set explicitly; omitempty will drop it
			}
		}

		openAIResp.Choices = append(openAIResp.Choices, openAIChoice)
	}

	openAIResponseBytes, err := json.Marshal(openAIResp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAI response: %w", err)
	}

	return openAIResponseBytes, nil
}
