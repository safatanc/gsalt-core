package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransactionType string
type TransactionStatus string

const (
	TransactionTypeTopup             TransactionType = "TOPUP"
	TransactionTypeTransferIn        TransactionType = "TRANSFER_IN"
	TransactionTypeTransferOut       TransactionType = "TRANSFER_OUT"
	TransactionTypePayment           TransactionType = "PAYMENT"
	TransactionTypeWithdrawal        TransactionType = "WITHDRAWAL"
	TransactionTypeGiftIn            TransactionType = "GIFT_IN"
	TransactionTypeGiftOut           TransactionType = "GIFT_OUT"
	TransactionTypeVoucherRedemption TransactionType = "VOUCHER_REDEMPTION"

	TransactionStatusPending    TransactionStatus = "PENDING"
	TransactionStatusProcessing TransactionStatus = "PROCESSING"
	TransactionStatusCompleted  TransactionStatus = "COMPLETED"
	TransactionStatusFailed     TransactionStatus = "FAILED"
	TransactionStatusCancelled  TransactionStatus = "CANCELLED"
)

type Transaction struct {
	ID                    uuid.UUID         `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AccountID             uuid.UUID         `json:"account_id" gorm:"type:uuid;not null"`
	Type                  TransactionType   `json:"type" gorm:"type:transaction_type;not null"`
	Currency              string            `json:"currency" gorm:"type:varchar(5);not null"`
	Status                TransactionStatus `json:"status" gorm:"type:transaction_status;not null"`
	Description           *string           `json:"description" gorm:"type:text"`
	RelatedTransactionID  *uuid.UUID        `json:"related_transaction_id,omitempty" gorm:"type:uuid"`
	SourceAccountID       *uuid.UUID        `json:"source_account_id,omitempty" gorm:"type:uuid"`
	DestinationAccountID  *uuid.UUID        `json:"destination_account_id,omitempty" gorm:"type:uuid"`
	VoucherCode           *string           `json:"voucher_code,omitempty" gorm:"type:varchar(50)"`
	ExternalReferenceID   *string           `json:"external_reference_id,omitempty" gorm:"type:varchar(255)"`
	AmountGsaltUnits      int64             `json:"amount_gsalt_units" gorm:"type:bigint;not null"`
	ExchangeRateIDR       decimal.Decimal   `json:"exchange_rate_idr" gorm:"type:decimal(18,2);column:exchange_rate_idr"`
	PaymentAmount         *int64            `json:"payment_amount,omitempty" gorm:"type:bigint"`
	PaymentCurrency       *string           `json:"payment_currency,omitempty" gorm:"type:varchar(3)"`
	PaymentMethod         *string           `json:"payment_method,omitempty" gorm:"type:varchar(50)"`
	FeeGsaltUnits         int64             `json:"fee_gsalt_units" gorm:"type:bigint;not null"`
	TotalAmountGsaltUnits int64             `json:"total_amount_gsalt_units" gorm:"type:bigint;not null"`

	// Payment status fields
	PaymentStatus            PaymentStatus `json:"payment_status" gorm:"type:varchar(20);default:PENDING"`
	PaymentStatusDescription *string       `json:"payment_status_description" gorm:"type:text"`
	PaymentInitiatedAt       *time.Time    `json:"payment_initiated_at" gorm:"type:timestamp with time zone"`
	PaymentCompletedAt       *time.Time    `json:"payment_completed_at" gorm:"type:timestamp with time zone"`
	PaymentFailedAt          *time.Time    `json:"payment_failed_at" gorm:"type:timestamp with time zone"`
	PaymentExpiredAt         *time.Time    `json:"payment_expired_at" gorm:"type:timestamp with time zone"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at" gorm:"type:timestamp with time zone;autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"type:timestamp with time zone;autoUpdateTime"`
	CompletedAt *time.Time `json:"completed_at,omitempty" gorm:"type:timestamp with time zone"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" gorm:"type:timestamp with time zone;index"`

	// Relations (not stored in DB)
	PaymentDetails *PaymentDetails `json:"payment_details,omitempty" gorm:"-"`
}

// Request/Response structs
type TopupRequest struct {
	AmountGsalt         string                       `json:"amount_gsalt" validate:"required,numeric,gt=0"`
	PaymentAmount       *int64                       `json:"payment_amount,omitempty" validate:"omitempty,gt=0"`
	PaymentCurrency     *string                      `json:"payment_currency,omitempty" validate:"omitempty,len=3"`
	PaymentMethod       *string                      `json:"payment_method,omitempty" validate:"omitempty,max=50"`
	ExternalReferenceID *string                      `json:"external_reference_id,omitempty" validate:"omitempty,max=255"`
	PaymentDetails      *PaymentDetailsCreateRequest `json:"payment_details,omitempty"`
}

type TransferRequest struct {
	DestinationAccountID string  `json:"destination_account_id" validate:"required,uuid"`
	AmountGsalt          string  `json:"amount_gsalt" validate:"required,numeric,gt=0"`
	Description          *string `json:"description,omitempty" validate:"omitempty,max=500"`
}

type PaymentRequest struct {
	AccountID        string                       `json:"account_id" validate:"required,uuid"`
	AmountGsaltUnits int64                        `json:"amount_gsalt_units" validate:"required,min=1"`
	PaymentMethod    string                       `json:"payment_method" validate:"required,max=50"`
	CustomerName     *string                      `json:"customer_name,omitempty" validate:"omitempty,max=255"`
	CustomerEmail    *string                      `json:"customer_email,omitempty" validate:"omitempty,email"`
	CustomerPhone    *string                      `json:"customer_phone,omitempty" validate:"omitempty,max=20"`
	CustomerAddress  *string                      `json:"customer_address,omitempty" validate:"omitempty,max=500"`
	RedirectURL      *string                      `json:"redirect_url,omitempty" validate:"omitempty,url"`
	PaymentDetails   *PaymentDetailsCreateRequest `json:"payment_details,omitempty"`
}

type WithdrawalRequest struct {
	AmountGsalt         string  `json:"amount_gsalt" validate:"required,numeric,gt=0"`
	BankCode            string  `json:"bank_code" validate:"required,max=10"`
	AccountNumber       string  `json:"account_number" validate:"required,max=50"`
	RecipientName       string  `json:"recipient_name" validate:"required,max=255"`
	Description         *string `json:"description,omitempty" validate:"omitempty,max=500"`
	ExternalReferenceID *string `json:"external_reference_id,omitempty" validate:"omitempty,max=255"`
}

type BankListResponse struct {
	BankCode    string `json:"bank_code"`
	BankName    string `json:"bank_name"`
	Available   bool   `json:"available"`
	MinAmount   int64  `json:"min_amount"`
	MaxAmount   int64  `json:"max_amount"`
	Fee         int64  `json:"fee"`
	Maintenance bool   `json:"maintenance"`
}

type WithdrawalResponse struct {
	Transaction    *Transaction `json:"transaction"`
	DisbursementID *string      `json:"disbursement_id,omitempty"`
	EstimatedTime  string       `json:"estimated_time"`
	Status         string       `json:"status"`
}

// Add missing request/response structs
type TransactionCreateRequest struct {
	AccountID            string           `json:"account_id" validate:"required,uuid"`
	Type                 TransactionType  `json:"type" validate:"required,oneof=TOPUP TRANSFER_IN TRANSFER_OUT PAYMENT GIFT_IN GIFT_OUT WITHDRAWAL"`
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

type TransactionUpdateRequest struct {
	Status          *TransactionStatus `json:"status,omitempty" validate:"omitempty,oneof=PENDING PROCESSING COMPLETED FAILED CANCELLED"`
	ExchangeRateIDR *decimal.Decimal   `json:"exchange_rate_idr,omitempty" validate:"omitempty,gt=0"`
	PaymentAmount   *int64             `json:"payment_amount,omitempty" validate:"omitempty,gt=0"`
	PaymentCurrency *string            `json:"payment_currency,omitempty" validate:"omitempty,len=3"`
	PaymentMethod   *string            `json:"payment_method,omitempty" validate:"omitempty,max=50"`
	Description     *string            `json:"description,omitempty" validate:"omitempty,max=500"`
}

type TransactionResponse struct {
	Transaction    *Transaction    `json:"transaction"`
	PaymentDetails *PaymentDetails `json:"payment_details,omitempty"`
}
