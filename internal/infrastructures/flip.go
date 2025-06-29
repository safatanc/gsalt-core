package infrastructures

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

type FlipConfig struct {
	SecretKey    string
	Environment  string // "sandbox" or "production"
	BaseURL      string
	WebhookToken string
	CallbackURL  string
	SuccessURL   string
	FailureURL   string
}

type FlipClient struct {
	HTTPClient *http.Client
	Config     *FlipConfig
	BaseURL    string
	AuthHeader string
}

// NewFlipClient creates a new Flip HTTP client with configuration
func NewFlipClient() *FlipClient {
	config := &FlipConfig{
		SecretKey:   Config.FlipConfig.SecretKey,
		Environment: Config.FlipConfig.Environment,
	}

	// Set base URL based on environment for V3 API
	if config.Environment == "production" {
		config.BaseURL = "https://bigflip.id/api/v3"
	} else {
		config.BaseURL = "https://bigflip.id/big_sandbox_api/v3"
	}

	// Create basic auth header
	authString := base64.StdEncoding.EncodeToString([]byte(config.SecretKey + ":"))
	authHeader := "Basic " + authString

	return &FlipClient{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Config:     config,
		BaseURL:    config.BaseURL,
		AuthHeader: authHeader,
	}
}

// GetAuthHeader returns the properly formatted authorization header
func (c *FlipClient) GetAuthHeader() string {
	return c.AuthHeader
}

// GetBaseURL returns the base URL for the current environment
func (c *FlipClient) GetBaseURL() string {
	return c.BaseURL
}

// GetFullURL constructs the full URL for an endpoint
func (c *FlipClient) GetFullURL(endpoint string) string {
	return fmt.Sprintf("%s%s", c.BaseURL, endpoint)
}
