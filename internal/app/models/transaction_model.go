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
	ID                    uuid.UUID         `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AccountID             uuid.UUID         `json:"account_id" gorm:"type:uuid;not null;index"`
	Type                  TransactionType   `json:"type" gorm:"type:varchar(20);not null"`
	AmountGsaltUnits      int64             `json:"amount_gsalt_units" gorm:"not null"`
	FeeGsaltUnits         int64             `json:"fee_gsalt_units" gorm:"default:0"`
	TotalAmountGsaltUnits int64             `json:"total_amount_gsalt_units" gorm:"not null"`
	Currency              string            `json:"currency" gorm:"type:varchar(10);default:'GSALT'"`
	ExchangeRateIDR       *decimal.Decimal  `json:"exchange_rate_idr,omitempty" gorm:"column:exchange_rate_idr;type:decimal(20,4)"`
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
func (t *Transaction) GetPaymentInstructions() (*PaymentGatewayResponse, error) {
	if t.PaymentInstructions == nil || *t.PaymentInstructions == "" {
		return nil, nil
	}

	var instructions PaymentGatewayResponse
	if err := json.Unmarshal([]byte(*t.PaymentInstructions), &instructions); err != nil {
		return nil, err
	}

	return &instructions, nil
}

// TopupResponse represents the response for topup operations
type TransactionResponse struct {
	Transaction         *Transaction            `json:"transaction"`
	PaymentInstructions *PaymentGatewayResponse `json:"payment_instructions,omitempty"`
}

type TransactionCreateRequest struct {
	AccountID            string           `json:"account_id" validate:"required,uuid"`
	Type                 TransactionType  `json:"type" validate:"required,oneof=TOPUP TRANSFER_IN TRANSFER_OUT PAYMENT GIFT_IN GIFT_OUT VOUCHER_REDEMPTION WITHDRAWAL"`
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

// Helper method for CreateRequest to set GSALT amount from decimal
func (req *TransactionCreateRequest) SetGsaltAmountFromDecimal(amount decimal.Decimal) {
	req.AmountGsaltUnits = amount.Mul(decimal.NewFromInt(100)).IntPart()
}

// Helper method for CreateRequest to get GSALT amount as decimal
func (req *TransactionCreateRequest) GetGsaltAmountAsDecimal() decimal.Decimal {
	return decimal.NewFromInt(req.AmountGsaltUnits).Div(decimal.NewFromInt(100))
}

type TransactionUpdateRequest struct {
	Status               *TransactionStatus `json:"status,omitempty" validate:"omitempty,oneof=PENDING PROCESSING COMPLETED FAILED CANCELLED"`
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

// Request models for transaction operations
type TopupRequest struct {
	AmountGsalt         string  `json:"amount_gsalt" validate:"required,numeric,gt=0"`
	PaymentAmount       *int64  `json:"payment_amount,omitempty" validate:"omitempty,gt=0"`
	PaymentCurrency     *string `json:"payment_currency,omitempty" validate:"omitempty,len=3"`
	PaymentMethod       *string `json:"payment_method,omitempty" validate:"omitempty,max=50"`
	ExternalReferenceID *string `json:"external_reference_id,omitempty" validate:"omitempty,max=255"`
}

type TransferRequest struct {
	DestinationAccountID string  `json:"destination_account_id" validate:"required,uuid"`
	AmountGsalt          string  `json:"amount_gsalt" validate:"required,numeric,gt=0"`
	Description          *string `json:"description,omitempty" validate:"omitempty,max=500"`
}

// PaymentRequest represents a payment request with comprehensive payment method support
type PaymentRequest struct {
	AccountID          string  `json:"account_id" validate:"required,uuid"`
	AmountGsaltUnits   int64   `json:"amount_gsalt_units" validate:"required,min=1"`
	PaymentMethod      string  `json:"payment_method" validate:"required,max=50"`
	CustomerName       *string `json:"customer_name,omitempty" validate:"omitempty,max=255"`
	CustomerEmail      *string `json:"customer_email,omitempty" validate:"omitempty,email"`
	CustomerPhone      *string `json:"customer_phone,omitempty" validate:"omitempty,max=20"`
	CustomerAddress    *string `json:"customer_address,omitempty" validate:"omitempty,max=500"`
	RedirectURL        *string `json:"redirect_url,omitempty" validate:"omitempty,url"`
	VABank             *string `json:"va_bank,omitempty" validate:"omitempty,max=20"`
	EWalletProvider    *string `json:"ewallet_provider,omitempty" validate:"omitempty,max=20"`
	CreditCardProvider *string `json:"credit_card_provider,omitempty" validate:"omitempty,max=20"`
	RetailProvider     *string `json:"retail_provider,omitempty" validate:"omitempty,max=20"`
	DirectDebitBank    *string `json:"direct_debit_bank,omitempty" validate:"omitempty,max=20"`
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
	ExternalPaymentID *string `json:"external_payment_id,omitempty" validate:"omitempty,max=255"`
}

type RejectPaymentRequest struct {
	Reason *string `json:"reason,omitempty" validate:"omitempty,max=500"`
}

// WithdrawalRequest represents a request to withdraw GSALT to bank account
type WithdrawalRequest struct {
	AmountGsalt         string  `json:"amount_gsalt" validate:"required,numeric,gt=0"`
	BankCode            string  `json:"bank_code" validate:"required,max=10"`
	AccountNumber       string  `json:"account_number" validate:"required,max=50"`
	RecipientName       string  `json:"recipient_name" validate:"required,max=255"`
	Description         *string `json:"description,omitempty" validate:"omitempty,max=500"`
	ExternalReferenceID *string `json:"external_reference_id,omitempty" validate:"omitempty,max=255"`
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

type GiftRequest struct {
	DestinationAccountID string  `json:"destination_account_id" validate:"required,uuid"`
	AmountGsalt          string  `json:"amount_gsalt" validate:"required,numeric,gt=0"`
	Description          *string `json:"description,omitempty" validate:"omitempty,max=500"`
}
