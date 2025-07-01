package apikey

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Scope represents an API key permission scope
type Scope string

// Available scopes
const (
	ScopeRead       Scope = "READ"
	ScopeWrite      Scope = "WRITE"
	ScopePayment    Scope = "PAYMENT"
	ScopeWithdrawal Scope = "WITHDRAWAL"
	ScopeAdmin      Scope = "ADMIN"
)

// APIKey represents a merchant API key
type APIKey struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	MerchantID uuid.UUID  `json:"merchant_id" db:"merchant_id"`
	KeyName    string     `json:"key_name" db:"key_name"`
	APIKey     string     `json:"api_key" db:"api_key"`
	Prefix     string     `json:"prefix" db:"prefix"`
	Scopes     []Scope    `json:"scopes" db:"scopes"`
	RateLimit  int        `json:"rate_limit" db:"rate_limit"`
	LastUsedAt *time.Time `json:"last_used_at" db:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at" db:"deleted_at"`
}

// APIKeyUsage represents an API key usage record
type APIKeyUsage struct {
	ID         uuid.UUID `json:"id" db:"id"`
	APIKeyID   uuid.UUID `json:"api_key_id" db:"api_key_id"`
	Endpoint   string    `json:"endpoint" db:"endpoint"`
	Method     string    `json:"method" db:"method"`
	IPAddress  string    `json:"ip_address" db:"ip_address"`
	UserAgent  string    `json:"user_agent" db:"user_agent"`
	StatusCode int       `json:"status_code" db:"status_code"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// GenerateAPIKey generates a new API key with the given prefix
func GenerateAPIKey(prefix string) (string, error) {
	// Generate 20 random bytes
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode to base32 and remove padding
	encoded := base32.StdEncoding.EncodeToString(bytes)
	encoded = strings.ReplaceAll(encoded, "=", "")

	// Format as prefix_encoded
	return prefix + "_" + encoded, nil
}

// ValidateScope checks if a scope is valid
func ValidateScope(scope Scope) bool {
	switch scope {
	case ScopeRead, ScopeWrite, ScopePayment, ScopeWithdrawal, ScopeAdmin:
		return true
	default:
		return false
	}
}

// HasScope checks if an API key has a specific scope
func (k *APIKey) HasScope(scope Scope) bool {
	for _, s := range k.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// IsExpired checks if an API key is expired
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsActive checks if an API key is active (not deleted and not expired)
func (k *APIKey) IsActive() bool {
	return k.DeletedAt == nil && !k.IsExpired()
}
