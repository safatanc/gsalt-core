package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type PaymentMethod struct {
	ID                       *uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name                     string          `json:"name" gorm:"type:varchar(255);not null"`
	Code                     string          `json:"code" gorm:"type:varchar(50);not null"`
	Currency                 string          `json:"currency" gorm:"type:varchar(3);not null"`
	MethodType               string          `json:"method_type" gorm:"type:varchar(50);not null"`
	ProviderCode             string          `json:"provider_code" gorm:"type:varchar(50);not null"`
	ProviderMethodCode       string          `json:"provider_method_code" gorm:"type:varchar(50);not null"`
	ProviderMethodType       string          `json:"provider_method_type" gorm:"type:varchar(50);not null"`
	PaymentFeeFlat           int64           `json:"payment_fee_flat" gorm:"default:0"`
	PaymentFeePercent        decimal.Decimal `json:"payment_fee_percent" gorm:"type:decimal(8,5);default:0.0"`
	WithdrawalFeeFlat        int64           `json:"withdrawal_fee_flat" gorm:"default:0"`
	WithdrawalFeePercent     decimal.Decimal `json:"withdrawal_fee_percent" gorm:"type:decimal(8,5);default:0.0"`
	IsActive                 bool            `json:"is_active" gorm:"default:true"`
	IsAvailableForTopup      bool            `json:"is_available_for_topup" gorm:"default:true"`
	IsAvailableForWithdrawal bool            `json:"is_available_for_withdrawal" gorm:"default:false"`
	CreatedAt                *time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt                *time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt                *gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// PaymentMethodFilter defines the query parameters for filtering payment methods.
type PaymentMethodFilter struct {
	Currency                 *string `query:"currency" validate:"omitempty,len=3,uppercase"`
	IsActive                 *bool   `query:"is_active"`
	IsAvailableForTopup      *bool   `query:"is_available_for_topup"`
	IsAvailableForWithdrawal *bool   `query:"is_available_for_withdrawal"`
}

// PaymentMethodResponse defines the structured response for a payment method.
// It contains all necessary fields for the client without exposing provider-specific details.
type PaymentMethodResponse struct {
	ID                       uuid.UUID       `json:"id"`
	Name                     string          `json:"name"`
	Code                     string          `json:"code"`
	Currency                 string          `json:"currency"`
	MethodType               string          `json:"method_type"`
	PaymentFeeFlat           int64           `json:"payment_fee_flat"`
	PaymentFeePercent        decimal.Decimal `json:"payment_fee_percent"`
	WithdrawalFeeFlat        int64           `json:"withdrawal_fee_flat"`
	WithdrawalFeePercent     decimal.Decimal `json:"withdrawal_fee_percent"`
	IsActive                 bool            `json:"is_active"`
	IsAvailableForTopup      bool            `json:"is_available_for_topup"`
	IsAvailableForWithdrawal bool            `json:"is_available_for_withdrawal"`
	CreatedAt                time.Time       `json:"created_at"`
	UpdatedAt                time.Time       `json:"updated_at"`
}
