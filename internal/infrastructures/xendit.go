package infrastructures

import (
	"net/http"
	"os"
	"time"
)

type XenditConfig struct {
	SecretKey    string
	WebhookToken string
	CallbackURL  string
	SuccessURL   string
	FailureURL   string
	Environment  string // "sandbox" or "live"
	BaseURL      string // Xendit API base URL
}

type XenditClient struct {
	HTTPClient *http.Client
	Config     XenditConfig
}

// NewXenditConfig creates XenditConfig from environment variables
func NewXenditConfig() XenditConfig {
	return XenditConfig{
		SecretKey:    getEnv("XENDIT_SECRET_KEY", ""),
		WebhookToken: getEnv("XENDIT_WEBHOOK_TOKEN", ""),
		CallbackURL:  getEnv("XENDIT_CALLBACK_URL", ""),
		SuccessURL:   getEnv("XENDIT_SUCCESS_URL", ""),
		FailureURL:   getEnv("XENDIT_FAILURE_URL", ""),
		Environment:  getEnv("XENDIT_ENVIRONMENT", "sandbox"),
		BaseURL:      getEnv("XENDIT_BASE_URL", "https://api.xendit.co"),
	}
}

// NewXenditClient creates a new Xendit HTTP client with configuration
func NewXenditClient(config XenditConfig) *XenditClient {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &XenditClient{
		HTTPClient: httpClient,
		Config:     config,
	}
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
