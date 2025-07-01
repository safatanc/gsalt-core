package middlewares

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/app/pkg"
	"github.com/safatanc/gsalt-core/internal/app/services"
)

// APIKeyMiddleware handles API key authentication and rate limiting
type APIKeyMiddleware struct {
	apiKeyService *services.MerchantAPIKeyService
	rateLimiter   RateLimiter
}

// NewAPIKeyMiddleware creates a new APIKeyMiddleware
func NewAPIKeyMiddleware(apiKeyService *services.MerchantAPIKeyService, rateLimiter RateLimiter) *APIKeyMiddleware {
	return &APIKeyMiddleware{
		apiKeyService: apiKeyService,
		rateLimiter:   rateLimiter,
	}
}

// AuthAPIKey creates a middleware that authenticates API key
func (m *APIKeyMiddleware) AuthAPIKey(c *fiber.Ctx) error {
	// Get API key from header
	key := c.Get("X-API-Key")
	if key == "" {
		// No API key, continue without merchant context
		return c.Next()
	}

	// Get API key from service
	apiKey, err := m.apiKeyService.GetAPIKey(c.Context(), key)
	if err != nil {
		return pkg.ErrorResponse(c, errors.NewUnauthorizedError("Invalid API key"))
	}

	// Check if API key is active
	if !apiKey.IsActive() {
		return pkg.ErrorResponse(c, errors.NewUnauthorizedError("API key is inactive or expired"))
	}

	// Check rate limit
	allowed, info := m.rateLimiter.Allow(
		"apikey:"+apiKey.ID.String(),
		Rate{
			Requests: apiKey.RateLimit,
			Window:   time.Minute,
		},
	)

	// Set rate limit headers
	c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
	c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", info.Remaining))
	c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", info.Reset.Unix()))

	if !allowed {
		return pkg.ErrorResponse(c, errors.NewTooManyRequestsError("Rate limit exceeded", info.Limit, info.Reset.Unix()))
	}

	// Check required scopes
	if !hasRequiredScopes(c, apiKey) {
		return pkg.ErrorResponse(c, errors.NewForbiddenError("Insufficient permissions"))
	}

	// Log API key usage asynchronously
	go func() {
		usage := &models.MerchantAPIKeyUsage{
			APIKeyID:   apiKey.ID,
			Endpoint:   c.Path(),
			Method:     c.Method(),
			IPAddress:  c.IP(),
			UserAgent:  c.Get("User-Agent"),
			StatusCode: c.Response().StatusCode(),
			CreatedAt:  time.Now(),
		}
		if err := m.apiKeyService.LogAPIKeyUsage(context.Background(), usage); err != nil {
			// TODO: Add proper error logging
		}
	}()

	// Add API key and merchant ID to locals
	c.Locals("api_key", apiKey)
	c.Locals("merchant_id", apiKey.MerchantID)

	return c.Next()
}

// hasRequiredScopes checks if the API key has all required scopes for the request
func hasRequiredScopes(c *fiber.Ctx, key *models.MerchantAPIKey) bool {
	// Define required scopes based on method and path
	var requiredScopes []models.APIKeyScope

	// Add READ scope for GET requests
	if c.Method() == fiber.MethodGet {
		requiredScopes = append(requiredScopes, models.APIKeyScopeRead)
	}

	// Add WRITE scope for POST/PUT/DELETE requests
	if c.Method() == fiber.MethodPost || c.Method() == fiber.MethodPut || c.Method() == fiber.MethodDelete {
		requiredScopes = append(requiredScopes, models.APIKeyScopeWrite)
	}

	// Add PAYMENT scope for payment endpoints
	if c.Path() == "/api/v1/payments" {
		requiredScopes = append(requiredScopes, models.APIKeyScopePayment)
	}

	// Add WITHDRAWAL scope for withdrawal endpoints
	if c.Path() == "/api/v1/withdrawals" {
		requiredScopes = append(requiredScopes, models.APIKeyScopeWithdrawal)
	}

	// Check if API key has all required scopes
	for _, scope := range requiredScopes {
		if !key.HasScope(scope) {
			return false
		}
	}

	return true
}
