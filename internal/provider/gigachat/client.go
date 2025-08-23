package gigachat

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/evgensoft/gigachat"
	"github.com/tidwall/sjson"
)

var (
	lastTimeRequest = time.Now()
	requestMutex    = sync.Mutex{}

	maxTimeoutTime = 1 * time.Second
	gigachatClient *gigachat.Client
)

func InitClient(token string) error {
	// Split the string into clientID and clientSecret
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("error in strings.SplitN: %v parts", len(parts))
	}

	clientID := parts[0]
	clientSecret := parts[1]

	// Create a client (ScopePersonal is used by default)
	gigachatClient = gigachat.NewClient(clientID, clientSecret)

	// If necessary, the scope can be changed
	gigachatClient.SetScope(gigachat.ScopeCorp)

	return nil
}

func Call(providerURL, model, token string, requestBody []byte) ([]byte, error) {
	// Execute requests in a single thread using mutex
	requestMutex.Lock()
	defer func() {
		// Remember the last request time
		lastTimeRequest = time.Now()
		requestMutex.Unlock()
	}()

	reqBody, err := sjson.SetBytes(requestBody, "model", model)
	if err != nil {
		return nil, fmt.Errorf("error in sjson.SetBytes: %w", err)
	}

	if !time.Now().After(lastTimeRequest.Add(maxTimeoutTime)) {
		log.Printf("Throttled %v seс for %s", time.Until(lastTimeRequest.Add(maxTimeoutTime)).Seconds(), model)
		time.Sleep(time.Until(lastTimeRequest.Add(maxTimeoutTime)))
	}

	resp, err := gigachatClient.SendBytes(reqBody)
	if err != nil {
		return nil, err
	}

	log.Printf("Resp from Gigachat - %s", resp)

	return ConvertGigaChatResponseToOpenAI(resp, model, true)
}

// GigaChatClient implements the Provider interface for Sberbank GigaChat
type GigaChatClient struct {
	ProviderURL string
	Model       string
	Token       string
}

// Call implements the Provider interface
func (c *GigaChatClient) Call(ctx context.Context, payload []byte, path string) ([]byte, error) {
	return Call(c.ProviderURL, c.Model, c.Token, payload)
}
