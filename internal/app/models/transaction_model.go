package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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
	TransactionTypeWithdrawal        TransactionType = "WITHDRAWAL"
)

type TransactionStatus string

const (
	TransactionStatusPending    TransactionStatus = "PENDING"
	TransactionStatusProcessing TransactionStatus = "PROCESSING"
	TransactionStatusCompleted  TransactionStatus = "COMPLETED"
	TransactionStatusFailed     TransactionStatus = "FAILED"
	TransactionStatusCancelled  TransactionStatus = "CANCELLED"
)

type Transaction struct {
	ID                    string            `json:"id" gorm:"primaryKey;type:varchar(36)"`
	AccountID             uuid.UUID         `json:"account_id" gorm:"type:uuid;not null;index"`
	Type                  TransactionType   `json:"type" gorm:"type:varchar(20);not null"`
	AmountGsaltUnits      int64             `json:"amount_gsalt_units" gorm:"not null"`
	FeeGsaltUnits         int64             `json:"fee_gsalt_units" gorm:"default:0"`
	TotalAmountGsaltUnits int64             `json:"total_amount_gsalt_units" gorm:"not null"`
	Currency              string            `json:"currency" gorm:"type:varchar(10);default:'GSALT'"`
	ExchangeRateIDR       *decimal.Decimal  `json:"exchange_rate_idr,omitempty" gorm:"type:decimal(20,4)"`
	PaymentAmount         *int64            `json:"payment_amount,omitempty"`
	PaymentCurrency       *string           `json:"payment_currency,omitempty" gorm:"type:varchar(10)"`
	PaymentMethod         *string           `json:"payment_method,omitempty" gorm:"type:varchar(50)"`
	Status                TransactionStatus `json:"status" gorm:"type:varchar(20);not null"`
	Description           *string           `json:"description,omitempty" gorm:"type:text"`
	RelatedTransactionID  *uuid.UUID        `json:"related_transaction_id,omitempty"`
	SourceAccountID       *uuid.UUID        `json:"source_account_id,omitempty"`
	DestinationAccountID  *uuid.UUID        `json:"destination_account_id,omitempty"`
	VoucherCode           *string           `json:"voucher_code,omitempty" gorm:"type:varchar(50)"`
	ExternalReferenceID   *string           `json:"external_reference_id,omitempty" gorm:"type:varchar(255)"`
	ExternalPaymentID     *string           `json:"external_payment_id,omitempty" gorm:"type:varchar(255)"`
	PaymentInstructions   *string           `json:"payment_instructions,omitempty" gorm:"type:text"`
	CreatedAt             time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt             time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
	CompletedAt           *time.Time        `json:"completed_at,omitempty"`
	Account               Account           `json:"account,omitempty" gorm:"foreignKey:AccountID"`
}

// Helper method to get GSALT amount as decimal (for display purposes)
func (t *Transaction) GetGsaltAmount() decimal.Decimal {
	return decimal.NewFromInt(t.AmountGsaltUnits).Div(decimal.NewFromInt(100))
}

// Helper method to set GSALT amount from decimal
func (t *Transaction) SetGsaltAmount(amount decimal.Decimal) {
	t.AmountGsaltUnits = amount.Mul(decimal.NewFromInt(100)).IntPart()
}

// Helper method to parse payment instructions JSON
func (t *Transaction) GetPaymentInstructionsAsMap() (map[string]interface{}, error) {
	if t.PaymentInstructions == nil {
		return nil, nil
	}

	var instructions map[string]interface{}
	if err := json.Unmarshal([]byte(*t.PaymentInstructions), &instructions); err != nil {
		return nil, err
	}

	return instructions, nil
}

// TopupResponse represents the response for topup operations
type TopupResponse struct {
	Transaction         *Transaction           `json:"transaction"`
	PaymentInstructions map[string]interface{} `json:"payment_instructions,omitempty"`
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
	PaymentInstructions  *string          `json:"payment_instructions,omitempty"`
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
	PaymentInstructions  *string            `json:"payment_instructions,omitempty"`
	Description          *string            `json:"description,omitempty" validate:"omitempty,max=500"`
	RelatedTransactionID *string            `json:"related_transaction_id,omitempty" validate:"omitempty,uuid"`
	SourceAccountID      *string            `json:"source_account_id,omitempty" validate:"omitempty,uuid"`
	DestinationAccountID *string            `json:"destination_account_id,omitempty" validate:"omitempty,uuid"`
	VoucherCode          *string            `json:"voucher_code,omitempty" validate:"omitempty,max=50"`
	ExternalReferenceID  *string            `json:"external_reference_id,omitempty" validate:"omitempty,max=255"`
	CompletedAt          *time.Time         `json:"completed_at,omitempty"`
}

// Request DTOs for transaction operations
type TopupRequest struct {
	AmountGsalt         string  `json:"amount_gsalt" validate:"required"`
	PaymentAmount       *int64  `json:"payment_amount,omitempty"`
	PaymentCurrency     *string `json:"payment_currency,omitempty"`
	PaymentMethod       *string `json:"payment_method,omitempty"`
	ExternalReferenceID *string `json:"external_reference_id,omitempty"`
}

type TransferRequest struct {
	DestinationAccountID string  `json:"destination_account_id" validate:"required,uuid"`
	AmountGsalt          string  `json:"amount_gsalt" validate:"required"`
	Description          *string `json:"description,omitempty"`
}

// PaymentRequest represents a payment request with comprehensive payment method support
type PaymentRequest struct {
	AccountID          string  `json:"account_id" validate:"required"`
	AmountGsaltUnits   int64   `json:"amount_gsalt_units" validate:"required,min=1"`
	PaymentMethod      string  `json:"payment_method" validate:"required"`
	CustomerName       *string `json:"customer_name,omitempty"`
	CustomerEmail      *string `json:"customer_email,omitempty"`
	CustomerPhone      *string `json:"customer_phone,omitempty"`
	CustomerAddress    *string `json:"customer_address,omitempty"`
	RedirectURL        *string `json:"redirect_url,omitempty"`
	VABank             *string `json:"va_bank,omitempty"`
	EWalletProvider    *string `json:"ewallet_provider,omitempty"`
	CreditCardProvider *string `json:"credit_card_provider,omitempty"`
	RetailProvider     *string `json:"retail_provider,omitempty"`
	DirectDebitBank    *string `json:"direct_debit_bank,omitempty"`
}

// PaymentResponse represents a payment response
type PaymentResponse struct {
	TransactionID       string     `json:"transaction_id"`
	Status              string     `json:"status"`
	PaymentMethod       string     `json:"payment_method"`
	Amount              int64      `json:"amount"`
	Fee                 int64      `json:"fee"`
	TotalAmount         int64      `json:"total_amount"`
	PaymentURL          string     `json:"payment_url,omitempty"`
	ExpiryTime          *time.Time `json:"expiry_time,omitempty"`
	PaymentInstructions string     `json:"payment_instructions,omitempty"`
}

type ConfirmPaymentRequest struct {
	ExternalPaymentID *string `json:"external_payment_id,omitempty"`
}

type RejectPaymentRequest struct {
	Reason *string `json:"reason,omitempty"`
}

// WithdrawalRequest represents a request to withdraw GSALT to bank account
type WithdrawalRequest struct {
	AmountGsalt         string  `json:"amount_gsalt" validate:"required"`
	BankCode            string  `json:"bank_code" validate:"required"`
	AccountNumber       string  `json:"account_number" validate:"required"`
	RecipientName       string  `json:"recipient_name" validate:"required"`
	Description         *string `json:"description,omitempty"`
	ExternalReferenceID *string `json:"external_reference_id,omitempty"`
}

// BankListResponse represents available banks for withdrawal
type BankListResponse struct {
	BankCode    string `json:"bank_code"`
	BankName    string `json:"bank_name"`
	Available   bool   `json:"available"`
	MinAmount   int64  `json:"min_amount"`
	MaxAmount   int64  `json:"max_amount"`
	Fee         int64  `json:"fee"`
	Maintenance bool   `json:"maintenance"`
}

// WithdrawalResponse represents withdrawal transaction response
type WithdrawalResponse struct {
	Transaction    *Transaction `json:"transaction"`
	DisbursementID *string      `json:"disbursement_id,omitempty"`
	EstimatedTime  string       `json:"estimated_time"`
	Status         string       `json:"status"`
}
