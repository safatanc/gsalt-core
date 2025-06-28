package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TransactionType string

const (
	TransactionTypeTopup             TransactionType = "TOPUP"
	TransactionTypeTransferIn        TransactionType = "TRANSFER_IN"
	TransactionTypeTransferOut       TransactionType = "TRANSFER_OUT"
	TransactionTypePayment           TransactionType = "PAYMENT"
	TransactionTypeGiftIn            TransactionType = "GIFT_IN"
	TransactionTypeGiftOut           TransactionType = "GIFT_OUT"
	TransactionTypeVoucherRedemption TransactionType = "VOUCHER_REDEMPTION"
)

type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "PENDING"
	TransactionStatusCompleted TransactionStatus = "COMPLETED"
	TransactionStatusFailed    TransactionStatus = "FAILED"
	TransactionStatusCancelled TransactionStatus = "CANCELLED"
)

type Transaction struct {
	ID                   uuid.UUID         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AccountID            uuid.UUID         `json:"account_id"`
	Type                 TransactionType   `json:"type"`
	AmountGsaltUnits     int64             `gorm:"column:amount_gsalt_units" json:"amount_gsalt_units"`
	Currency             string            `gorm:"default:GSALT" json:"currency"`
	ExchangeRateIDR      *decimal.Decimal  `gorm:"column:exchange_rate_idr;type:decimal(10,2);default:1000.00" json:"exchange_rate_idr,omitempty"`
	PaymentAmount        *int64            `gorm:"column:payment_amount" json:"payment_amount,omitempty"`
	PaymentCurrency      *string           `gorm:"column:payment_currency;size:5" json:"payment_currency,omitempty"`
	PaymentMethod        *string           `gorm:"column:payment_method;size:50" json:"payment_method,omitempty"`
	Status               TransactionStatus `json:"status"`
	Description          *string           `json:"description,omitempty"`
	RelatedTransactionID *uuid.UUID        `json:"related_transaction_id,omitempty"`
	SourceAccountID      *uuid.UUID        `json:"source_account_id,omitempty"`
	DestinationAccountID *uuid.UUID        `json:"destination_account_id,omitempty"`
	VoucherCode          *string           `json:"voucher_code,omitempty"`
	ExternalReferenceID  *string           `json:"external_reference_id,omitempty"`
	CreatedAt            time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
	CompletedAt          *time.Time        `json:"completed_at,omitempty"`
	DeletedAt            gorm.DeletedAt    `gorm:"index" json:"deleted_at"`
}

// Helper method to get GSALT amount as decimal (for display purposes)
func (t *Transaction) GetGsaltAmount() decimal.Decimal {
	return decimal.NewFromInt(t.AmountGsaltUnits).Div(decimal.NewFromInt(100))
}

// Helper method to set GSALT amount from decimal
func (t *Transaction) SetGsaltAmount(amount decimal.Decimal) {
	t.AmountGsaltUnits = amount.Mul(decimal.NewFromInt(100)).IntPart()
}

type TransactionCreateDto struct {
	AccountID            string           `json:"account_id" validate:"required,uuid"`
	Type                 TransactionType  `json:"type" validate:"required,oneof=TOPUP TRANSFER_IN TRANSFER_OUT PAYMENT GIFT_IN GIFT_OUT VOUCHER_REDEMPTION"`
	AmountGsaltUnits     int64            `json:"amount_gsalt_units" validate:"required,gt=0"`
	Currency             string           `json:"currency" validate:"omitempty,len=5" default:"GSALT"`
	ExchangeRateIDR      *decimal.Decimal `json:"exchange_rate_idr,omitempty" validate:"omitempty,gt=0"`
	PaymentAmount        *int64           `json:"payment_amount,omitempty" validate:"omitempty,gt=0"`
	PaymentCurrency      *string          `json:"payment_currency,omitempty" validate:"omitempty,len=3"`
	PaymentMethod        *string          `json:"payment_method,omitempty" validate:"omitempty,max=50"`
	Description          *string          `json:"description,omitempty" validate:"omitempty,max=500"`
	RelatedTransactionID *string          `json:"related_transaction_id,omitempty" validate:"omitempty,uuid"`
	SourceAccountID      *string          `json:"source_account_id,omitempty" validate:"omitempty,uuid"`
	DestinationAccountID *string          `json:"destination_account_id,omitempty" validate:"omitempty,uuid"`
	VoucherCode          *string          `json:"voucher_code,omitempty" validate:"omitempty,max=50"`
	ExternalReferenceID  *string          `json:"external_reference_id,omitempty" validate:"omitempty,max=255"`
}

// Helper method for CreateDto to set GSALT amount from decimal
func (dto *TransactionCreateDto) SetGsaltAmountFromDecimal(amount decimal.Decimal) {
	dto.AmountGsaltUnits = amount.Mul(decimal.NewFromInt(100)).IntPart()
}

// Helper method for CreateDto to get GSALT amount as decimal
func (dto *TransactionCreateDto) GetGsaltAmountAsDecimal() decimal.Decimal {
	return decimal.NewFromInt(dto.AmountGsaltUnits).Div(decimal.NewFromInt(100))
}

type TransactionUpdateDto struct {
	Status               *TransactionStatus `json:"status,omitempty" validate:"omitempty,oneof=PENDING COMPLETED FAILED CANCELLED"`
	ExchangeRateIDR      *decimal.Decimal   `json:"exchange_rate_idr,omitempty" validate:"omitempty,gt=0"`
	PaymentAmount        *int64             `json:"payment_amount,omitempty" validate:"omitempty,gt=0"`
	PaymentCurrency      *string            `json:"payment_currency,omitempty" validate:"omitempty,len=3"`
	PaymentMethod        *string            `json:"payment_method,omitempty" validate:"omitempty,max=50"`
	Description          *string            `json:"description,omitempty" validate:"omitempty,max=500"`
	RelatedTransactionID *string            `json:"related_transaction_id,omitempty" validate:"omitempty,uuid"`
	SourceAccountID      *string            `json:"source_account_id,omitempty" validate:"omitempty,uuid"`
	DestinationAccountID *string            `json:"destination_account_id,omitempty" validate:"omitempty,uuid"`
	VoucherCode          *string            `json:"voucher_code,omitempty" validate:"omitempty,max=50"`
	ExternalReferenceID  *string            `json:"external_reference_id,omitempty" validate:"omitempty,max=255"`
	CompletedAt          *time.Time         `json:"completed_at,omitempty"`
}
