package config

import (
	"embed"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"ai-proxy/internal/errors"
	"ai-proxy/internal/provider/gigachat"
	"ai-proxy/internal/schema"

	"gopkg.in/yaml.v3"
)

type RateLimit struct {
	MinuteCount int
	HourCount   int
	DayCount    int
	LastMinute  time.Time
	LastHour    time.Time
	LastDay     time.Time
	LastRequest time.Time
	mux         sync.Mutex // Fine-grained locking for each rate limit
}

// Lock acquires the mutex for this rate limit
func (rl *RateLimit) Lock() {
	rl.mux.Lock()
}

// Unlock releases the mutex for this rate limit
func (rl *RateLimit) Unlock() {
	rl.mux.Unlock()
}

// GetMinuteCount returns the current minute count
func (rl *RateLimit) GetMinuteCount() int {
	return rl.MinuteCount
}

// GetHourCount returns the current hour count
func (rl *RateLimit) GetHourCount() int {
	return rl.HourCount
}

// GetDayCount returns the current day count
func (rl *RateLimit) GetDayCount() int {
	return rl.DayCount
}

// SetCounts sets the minute, hour, and day counts
func (rl *RateLimit) SetCounts(minute, hour, day int) {
	rl.MinuteCount = minute
	rl.HourCount = hour
	rl.DayCount = day
}

// GetLastTimes returns the last minute, hour, and day times
func (rl *RateLimit) GetLastTimes() (time.Time, time.Time, time.Time) {
	return rl.LastMinute, rl.LastHour, rl.LastDay
}

// SetLastTimes sets the last minute, hour, and day times
func (rl *RateLimit) SetLastTimes(minute, hour, day time.Time) {
	rl.LastMinute = minute
	rl.LastHour = hour
	rl.LastDay = day
}

// GetLastRequest returns the last request time
func (rl *RateLimit) GetLastRequest() time.Time {
	return rl.LastRequest
}

// SetLastRequest sets the last request time
func (rl *RateLimit) SetLastRequest(lastRequest time.Time) {
	rl.LastRequest = lastRequest
}

type Model struct {
	Name             string                  `yaml:"name"`
	Provider         string                  `yaml:"provider"`
	Priority         int                     `yaml:"priority"`
	RequestsPerMin   int                     `yaml:"requests_per_minute"`
	RequestsPerHour  int                     `yaml:"requests_per_hour"`
	RequestsPerDay   int                     `yaml:"requests_per_day"`
	URL              string                  `yaml:"url"`
	Token            string                  `yaml:"token"`
	MaxRequestLength int                     `yaml:"max_request_length"`
	Size             string                  `yaml:"model_size"`
	HttpClientConfig schema.HttpClientConfig `yaml:"http_client_config" json:"http_client_config"`
}

// Config represents the configuration structure for the proxy
type Config struct {
	Models []Model `yaml:"models"`
}

var (
	RateLimits map[string]*RateLimit
	Models     []Model
	// Note: Models and RateLimits are only modified during initialization in main.go
	// and then only read during operation, so they don't need synchronization.
	// If they were to be modified during operation, we would need to add mutexes.
)

// LoadConfig loads the configuration from either a file or embedded config
func LoadConfig(configFile embed.FS, configPath string) (*Config, error) {
	var config Config
	var cfgBytes []byte
	var err error

	if configPath != "" {
		cfgBytes, err = os.ReadFile(configPath)
	} else {
		cfgBytes, err = configFile.ReadFile("provider_config.yaml")
	}
	if err != nil {
		proxyErr := errors.NewConfigurationError(
			fmt.Sprintf("Error reading config file: %v", err), err)
		return nil, proxyErr
	}
	if err := yaml.Unmarshal(cfgBytes, &config); err != nil {
		proxyErr := errors.NewConfigurationError(
			fmt.Sprintf("Error Unmarshal config: %v", err), err)
		return nil, proxyErr
	}

	return &config, nil
}

// Initialize initializes the models and rate limits from the configuration
func (c *Config) Initialize() error {
	// Initialize global variables
	RateLimits = make(map[string]*RateLimit)
	Models = make([]Model, 0, len(c.Models))

	for _, v := range c.Models {
		Models = append(Models, v)
		RateLimits[v.Name] = &RateLimit{}

		if v.Provider == "gigachat" {
			if err := gigachat.InitClient(v.Token); err != nil {
				return fmt.Errorf("failed to initialize gigachat client: %w", err)
			}
		}

		// Log model loading
		// log.Println("Load model ", v.Name)
	}

	return nil
}

// InitializeModels initializes the models and rate limits from the configuration
func InitializeModels(config *Config) error {
	return config.Initialize()
}

// LoadRetryConfig loads retry configuration from environment variables
func LoadRetryConfig(envFile map[string]string) (maxRetries int, baseDelay, maxDelay time.Duration, jitter float64) {
	maxRetries = 3
	baseDelay = 1 * time.Second
	maxDelay = 30 * time.Second
	jitter = 0.1

	if maxRetriesStr, exists := envFile["RETRY_MAX_RETRIES"]; exists {
		if maxRetriesVal, err := strconv.Atoi(maxRetriesStr); err == nil {
			maxRetries = maxRetriesVal
		}
	}

	if baseDelayStr, exists := envFile["RETRY_BASE_DELAY"]; exists {
		if baseDelayVal, err := time.ParseDuration(baseDelayStr); err == nil {
			baseDelay = baseDelayVal
		}
	}

	if maxDelayStr, exists := envFile["RETRY_MAX_DELAY"]; exists {
		if maxDelayVal, err := time.ParseDuration(maxDelayStr); err == nil {
			maxDelay = maxDelayVal
		}
	}

	if jitterStr, exists := envFile["RETRY_JITTER"]; exists {
		if jitterVal, err := strconv.ParseFloat(jitterStr, 64); err == nil {
			jitter = jitterVal
		}
	}

	return maxRetries, baseDelay, maxDelay, jitter
}
