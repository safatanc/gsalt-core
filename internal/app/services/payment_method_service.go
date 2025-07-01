package services

import (
	"fmt"
	"time"

	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/infrastructures"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// PaymentMethodService provides business logic for payment methods.
type PaymentMethodService struct {
	db        *gorm.DB
	validator *infrastructures.Validator
}

// NewPaymentMethodService creates a new instance of PaymentMethodService.
func NewPaymentMethodService(db *gorm.DB, validator *infrastructures.Validator) *PaymentMethodService {
	return &PaymentMethodService{
		db:        db,
		validator: validator,
	}
}

// GetPaymentMethods retrieves a list of payment methods based on a filter.
func (s *PaymentMethodService) GetPaymentMethods(filter *models.PaymentMethodFilter) ([]models.PaymentMethodResponse, error) {
	if err := s.validator.Validate(filter); err != nil {
		return nil, err
	}

	methods, err := s.find(filter)
	if err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to retrieve payment methods")
	}

	var response []models.PaymentMethodResponse
	for _, method := range methods {
		response = append(response, s.toPaymentMethodResponse(method))
	}

	return response, nil
}

// FindByCode finds a single active payment method by its code.
func (s *PaymentMethodService) FindByCode(code string) (*models.PaymentMethod, error) {
	var method models.PaymentMethod
	err := s.db.Where("code = ? AND is_active = ?", code, true).First(&method).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewBadRequestError(fmt.Sprintf("Payment method with code '%s' not found or is not active", code))
		}
		return nil, errors.NewInternalServerError(err, "Database error while finding payment method")
	}
	return &method, nil
}

// CalculateFee calculates the transaction fee based on the payment method's configuration.
// The amount is in the fee's currency (e.g., IDR for Flip).
func (s *PaymentMethodService) CalculateFee(method models.PaymentMethod, amount decimal.Decimal) decimal.Decimal {
	fee := decimal.NewFromInt(method.PaymentFeeFlat)
	percentageFee := amount.Mul(method.PaymentFeePercent)
	return fee.Add(percentageFee)
}

// toPaymentMethodResponse converts a PaymentMethod model to a PaymentMethodResponse.
func (s *PaymentMethodService) toPaymentMethodResponse(method models.PaymentMethod) models.PaymentMethodResponse {
	// Ensure pointer values are handled safely
	var createdAt, updatedAt time.Time
	if method.CreatedAt != nil {
		createdAt = *method.CreatedAt
	}
	if method.UpdatedAt != nil {
		updatedAt = *method.UpdatedAt
	}

	return models.PaymentMethodResponse{
		ID:                       *method.ID,
		Name:                     method.Name,
		Code:                     method.Code,
		Currency:                 method.Currency,
		MethodType:               method.MethodType,
		PaymentFeeFlat:           method.PaymentFeeFlat,
		PaymentFeePercent:        method.PaymentFeePercent,
		WithdrawalFeeFlat:        method.WithdrawalFeeFlat,
		WithdrawalFeePercent:     method.WithdrawalFeePercent,
		IsActive:                 method.IsActive,
		IsAvailableForTopup:      method.IsAvailableForTopup,
		IsAvailableForWithdrawal: method.IsAvailableForWithdrawal,
		CreatedAt:                createdAt,
		UpdatedAt:                updatedAt,
	}
}

// find retrieves payment methods from the database based on a filter.
func (s *PaymentMethodService) find(filter *models.PaymentMethodFilter) ([]models.PaymentMethod, error) {
	var methods []models.PaymentMethod
	query := s.db.Model(&models.PaymentMethod{})

	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	} else {
		query = query.Where("is_active = ?", true) // Default to active methods
	}

	if filter.Currency != nil {
		query = query.Where("currency = ?", *filter.Currency)
	}
	if filter.IsAvailableForTopup != nil {
		query = query.Where("is_available_for_topup = ?", *filter.IsAvailableForTopup)
	}
	if filter.IsAvailableForWithdrawal != nil {
		query = query.Where("is_available_for_withdrawal = ?", *filter.IsAvailableForWithdrawal)
	}

	err := query.Order("name ASC").Find(&methods).Error
	if err != nil {
		return nil, err
	}
	return methods, nil
}
