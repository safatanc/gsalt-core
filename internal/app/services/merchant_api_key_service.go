package services

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"gorm.io/gorm"
)

var (
	ErrInvalidAPIKey = errors.New("invalid API key")
	ErrAPIKeyExpired = errors.New("API key expired")
	ErrInvalidScope  = errors.New("invalid scope")
)

// MerchantAPIKeyService handles merchant API key operations
type MerchantAPIKeyService struct {
	db *gorm.DB
}

// NewMerchantAPIKeyService creates a new MerchantAPIKeyService
func NewMerchantAPIKeyService(db *gorm.DB) *MerchantAPIKeyService {
	return &MerchantAPIKeyService{db: db}
}

// CreateAPIKey creates a new API key for a merchant
func (s *MerchantAPIKeyService) CreateAPIKey(ctx context.Context, merchantID uuid.UUID, keyName string, scopes []models.APIKeyScope, rateLimit int, expiresAt *time.Time) (*models.MerchantAPIKey, error) {
	// Validate scopes
	for _, scope := range scopes {
		if !s.isValidScope(scope) {
			return nil, ErrInvalidScope
		}
	}

	// Generate API key
	apiKey, prefix, err := s.generateAPIKey()
	if err != nil {
		return nil, err
	}

	// Create API key record
	key := &models.MerchantAPIKey{
		MerchantID: merchantID,
		KeyName:    keyName,
		APIKey:     apiKey,
		Prefix:     prefix,
		Scopes:     scopes,
		RateLimit:  rateLimit,
		ExpiresAt:  expiresAt,
	}

	if err := s.db.WithContext(ctx).Create(key).Error; err != nil {
		return nil, err
	}

	return key, nil
}

// GetAPIKey gets an API key by its value
func (s *MerchantAPIKeyService) GetAPIKey(ctx context.Context, apiKey string) (*models.MerchantAPIKey, error) {
	var key models.MerchantAPIKey
	if err := s.db.WithContext(ctx).Where("api_key = ?", apiKey).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidAPIKey
		}
		return nil, err
	}

	if !key.IsActive() {
		return nil, ErrAPIKeyExpired
	}

	return &key, nil
}

// LogAPIKeyUsage logs API key usage
func (s *MerchantAPIKeyService) LogAPIKeyUsage(ctx context.Context, usage *models.MerchantAPIKeyUsage) error {
	return s.db.WithContext(ctx).Create(usage).Error
}

// ListAPIKeys lists all API keys for a merchant
func (s *MerchantAPIKeyService) ListAPIKeys(ctx context.Context, merchantID uuid.UUID) ([]models.MerchantAPIKey, error) {
	var keys []models.MerchantAPIKey
	if err := s.db.WithContext(ctx).Where("merchant_id = ?", merchantID).Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

// RevokeAPIKey revokes an API key
func (s *MerchantAPIKeyService) RevokeAPIKey(ctx context.Context, id uuid.UUID, merchantID uuid.UUID) error {
	now := time.Now()
	result := s.db.WithContext(ctx).
		Model(&models.MerchantAPIKey{}).
		Where("id = ? AND merchant_id = ?", id, merchantID).
		Update("deleted_at", now)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInvalidAPIKey
	}
	return nil
}

// UpdateAPIKey updates an API key's properties
func (s *MerchantAPIKeyService) UpdateAPIKey(ctx context.Context, id uuid.UUID, merchantID uuid.UUID, keyName string, scopes []models.APIKeyScope, rateLimit int, expiresAt *time.Time) (*models.MerchantAPIKey, error) {
	// Validate scopes
	for _, scope := range scopes {
		if !s.isValidScope(scope) {
			return nil, ErrInvalidScope
		}
	}

	var key models.MerchantAPIKey
	if err := s.db.WithContext(ctx).Where("id = ? AND merchant_id = ?", id, merchantID).First(&key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidAPIKey
		}
		return nil, err
	}

	// Update fields
	key.KeyName = keyName
	key.Scopes = scopes
	key.RateLimit = rateLimit
	key.ExpiresAt = expiresAt
	key.UpdatedAt = time.Now()

	if err := s.db.WithContext(ctx).Save(&key).Error; err != nil {
		return nil, err
	}

	return &key, nil
}

// generateAPIKey generates a new API key with prefix
func (s *MerchantAPIKeyService) generateAPIKey() (string, string, error) {
	prefix := "mk"
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}

	encoded := base32.StdEncoding.EncodeToString(bytes)
	encoded = strings.ReplaceAll(encoded, "=", "")

	return prefix + "_" + encoded, prefix, nil
}

// isValidScope checks if a scope is valid
func (s *MerchantAPIKeyService) isValidScope(scope models.APIKeyScope) bool {
	switch scope {
	case models.APIKeyScopeRead,
		models.APIKeyScopeWrite,
		models.APIKeyScopePayment,
		models.APIKeyScopeWithdrawal,
		models.APIKeyScopeAdmin:
		return true
	default:
		return false
	}
}
