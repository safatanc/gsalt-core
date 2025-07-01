package models

import (
	"time"

	"github.com/google/uuid"
)

// APIKeyScope represents an API key permission scope
type APIKeyScope string

// Available scopes
const (
	APIKeyScopeRead       APIKeyScope = "READ"
	APIKeyScopeWrite      APIKeyScope = "WRITE"
	APIKeyScopePayment    APIKeyScope = "PAYMENT"
	APIKeyScopeWithdrawal APIKeyScope = "WITHDRAWAL"
	APIKeyScopeAdmin      APIKeyScope = "ADMIN"
)

// MerchantAPIKey represents a merchant API key
type MerchantAPIKey struct {
	ID         uuid.UUID     `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MerchantID uuid.UUID     `json:"merchant_id" gorm:"type:uuid;not null"`
	KeyName    string        `json:"key_name" gorm:"type:varchar(100);not null"`
	APIKey     string        `json:"api_key" gorm:"type:varchar(255);not null;unique"`
	Prefix     string        `json:"prefix" gorm:"type:varchar(10);not null"`
	Scopes     []APIKeyScope `json:"scopes" gorm:"type:api_key_scope[];not null"`
	RateLimit  int           `json:"rate_limit" gorm:"not null;default:100"`
	LastUsedAt *time.Time    `json:"last_used_at"`
	ExpiresAt  *time.Time    `json:"expires_at"`
	CreatedAt  time.Time     `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt  time.Time     `json:"updated_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
	DeletedAt  *time.Time    `json:"deleted_at"`

	// Relations
	Merchant     Account               `json:"-" gorm:"foreignkey:MerchantID;references:ConnectID"`
	UsageHistory []MerchantAPIKeyUsage `json:"usage_history" gorm:"foreignkey:APIKeyID"`
}

// MerchantAPIKeyUsage represents an API key usage record
type MerchantAPIKeyUsage struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	APIKeyID   uuid.UUID `json:"api_key_id" gorm:"type:uuid;not null"`
	Endpoint   string    `json:"endpoint" gorm:"type:varchar(255);not null"`
	Method     string    `json:"method" gorm:"type:varchar(10);not null"`
	IPAddress  string    `json:"ip_address" gorm:"type:varchar(45);not null"`
	UserAgent  string    `json:"user_agent" gorm:"type:varchar(255)"`
	StatusCode int       `json:"status_code" gorm:"not null"`
	CreatedAt  time.Time `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`

	// Relations
	APIKey MerchantAPIKey `json:"-" gorm:"foreignkey:APIKeyID"`
}

// IsExpired checks if an API key is expired
func (k *MerchantAPIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsActive checks if an API key is active (not deleted and not expired)
func (k *MerchantAPIKey) IsActive() bool {
	return k.DeletedAt == nil && !k.IsExpired()
}

// HasScope checks if an API key has a specific scope
func (k *MerchantAPIKey) HasScope(scope APIKeyScope) bool {
	for _, s := range k.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// TableName returns the table name for GORM
func (MerchantAPIKey) TableName() string {
	return "merchant_api_keys"
}

// TableName returns the table name for GORM
func (MerchantAPIKeyUsage) TableName() string {
	return "merchant_api_key_usage"
}
