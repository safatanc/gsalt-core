package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/infrastructures"
	"gorm.io/gorm"
)

type PaymentService struct {
	db        *gorm.DB
	validator *infrastructures.Validator
}

func NewPaymentService(db *gorm.DB, validator *infrastructures.Validator) *PaymentService {
	return &PaymentService{
		db:        db,
		validator: validator,
	}
}

// CreatePaymentDetails creates a new payment details record
func (s *PaymentService) CreatePaymentDetails(ctx context.Context, req *models.PaymentDetailsCreateRequest) (*models.PaymentDetails, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	details := &models.PaymentDetails{
		ID:                   uuid.New(),
		TransactionID:        req.TransactionID,
		Provider:             req.Provider,
		ProviderPaymentID:    req.ProviderPaymentID,
		PaymentURL:           req.PaymentURL,
		QRCode:               req.QRCode,
		VirtualAccountNumber: req.VirtualAccountNumber,
		VirtualAccountBank:   req.VirtualAccountBank,
		RetailOutletCode:     req.RetailOutletCode,
		RetailPaymentCode:    req.RetailPaymentCode,
		CardToken:            req.CardToken,
		ExpiryTime:           req.ExpiryTime,
		ProviderFeeAmount:    req.ProviderFeeAmount,
		StatusHistory:        json.RawMessage("[]"),
		RawProviderResponse:  req.RawProviderResponse,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(details).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to create payment details")
	}

	return details, nil
}

// GetPaymentDetailsByTransactionID retrieves payment details for a transaction
func (s *PaymentService) GetPaymentDetailsByTransactionID(ctx context.Context, transactionID uuid.UUID) (*models.PaymentDetails, error) {
	var details models.PaymentDetails
	err := s.db.WithContext(ctx).
		Where("transaction_id = ?", transactionID).
		Order("created_at DESC").
		First(&details).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Payment details not found")
		}
		return nil, errors.NewInternalServerError(err, "Failed to get payment details")
	}

	return &details, nil
}

// UpdatePaymentStatus updates the payment status and details
func (s *PaymentService) UpdatePaymentStatus(ctx context.Context, transactionID uuid.UUID, req *models.PaymentStatusUpdateRequest) error {
	if err := s.validator.Validate(req); err != nil {
		return err
	}

	// Get existing payment details
	var details models.PaymentDetails
	err := s.db.WithContext(ctx).
		Where("transaction_id = ?", transactionID).
		First(&details).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewNotFoundError("Payment details not found")
		}
		return errors.NewInternalServerError(err, "Failed to get payment details")
	}

	// Update payment details
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if req.ProviderPaymentID != nil {
		updates["provider_payment_id"] = *req.ProviderPaymentID
	}
	if req.PaymentTime != nil {
		updates["payment_time"] = *req.PaymentTime
	}

	// Add status history
	var statusHistory []map[string]interface{}
	if details.StatusHistory != nil {
		if err := json.Unmarshal(details.StatusHistory, &statusHistory); err != nil {
			return errors.NewInternalServerError(err, "Failed to parse status history")
		}
	}

	statusHistory = append(statusHistory, map[string]interface{}{
		"status":      req.Status,
		"description": req.StatusDescription,
		"time":        time.Now(),
	})

	statusHistoryJSON, err := json.Marshal(statusHistory)
	if err != nil {
		return errors.NewInternalServerError(err, "Failed to marshal status history")
	}
	updates["status_history"] = statusHistoryJSON

	err = s.db.WithContext(ctx).
		Model(&details).
		Updates(updates).Error

	if err != nil {
		return errors.NewInternalServerError(err, "Failed to update payment details")
	}

	return nil
}

// CreatePaymentAccount creates a new payment account
func (s *PaymentService) CreatePaymentAccount(ctx context.Context, req *models.PaymentAccountCreateRequest) (*models.PaymentAccount, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	account := &models.PaymentAccount{
		ID:            uuid.New(),
		AccountID:     req.AccountID,
		Type:          req.Type,
		Provider:      req.Provider,
		AccountNumber: req.AccountNumber,
		AccountName:   req.AccountName,
		IsVerified:    false,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(account).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to create payment account")
	}

	return account, nil
}

// GetPaymentAccounts retrieves all payment accounts for a user
func (s *PaymentService) GetPaymentAccounts(ctx context.Context, accountID uuid.UUID) ([]*models.PaymentAccount, error) {
	var accounts []*models.PaymentAccount
	err := s.db.WithContext(ctx).
		Where("account_id = ? AND is_active = ?", accountID, true).
		Order("created_at DESC").
		Find(&accounts).Error

	if err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to get payment accounts")
	}

	return accounts, nil
}

// VerifyPaymentAccount marks a payment account as verified
func (s *PaymentService) VerifyPaymentAccount(ctx context.Context, accountID uuid.UUID) error {
	now := time.Now()
	result := s.db.WithContext(ctx).
		Model(&models.PaymentAccount{}).
		Where("id = ?", accountID).
		Updates(map[string]interface{}{
			"is_verified":       true,
			"verification_time": now,
			"updated_at":        now,
		})

	if result.Error != nil {
		return errors.NewInternalServerError(result.Error, "Failed to verify payment account")
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("Payment account not found")
	}

	return nil
}

// DeactivatePaymentAccount soft deletes a payment account
func (s *PaymentService) DeactivatePaymentAccount(ctx context.Context, accountID uuid.UUID) error {
	now := time.Now()
	result := s.db.WithContext(ctx).
		Model(&models.PaymentAccount{}).
		Where("id = ?", accountID).
		Updates(map[string]interface{}{
			"is_active":  false,
			"updated_at": now,
		})

	if result.Error != nil {
		return errors.NewInternalServerError(result.Error, "Failed to deactivate payment account")
	}

	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("Payment account not found")
	}

	return nil
}
