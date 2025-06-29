package infrastructures

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type FlipConfig struct {
	SecretKey          string
	Environment        string // "sandbox" or "production"
	BaseURL            string
	WebhookToken       string
	CallbackURL        string
	SuccessURL         string
	FailureURL         string
	DefaultRedirectURL string
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

	// Set base URL based on environment - No version, let each endpoint handle its own version
	if config.Environment == "production" {
		config.BaseURL = "https://bigflip.id/api"
	} else {
		config.BaseURL = "https://bigflip.id/big_sandbox_api"
	}

	// Create basic auth header - Flip requires secret key with colon
	secretKey := config.SecretKey
	if secretKey != "" && !strings.HasSuffix(secretKey, ":") {
		secretKey += ":" // Add colon if not present
	}
	authString := base64.StdEncoding.EncodeToString([]byte(secretKey))
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

// GetDefaultRedirectURL returns the default redirect URL from configuration
func (c *FlipClient) GetDefaultRedirectURL() string {
	if Config != nil && Config.FlipConfig != nil && Config.FlipConfig.DefaultRedirectURL != "" {
		return Config.FlipConfig.DefaultRedirectURL
	}
	return "https://gsalt.com/payment/success" // Fallback default
}
