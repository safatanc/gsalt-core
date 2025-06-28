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
	Amount               decimal.Decimal   `gorm:"type:decimal(18,2)" json:"amount"`
	Currency             string            `json:"currency"`
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

type TransactionCreateDto struct {
	AccountID            string          `json:"account_id" validate:"required,uuid"`
	Type                 TransactionType `json:"type" validate:"required,oneof=TOPUP TRANSFER_IN TRANSFER_OUT PAYMENT GIFT_IN GIFT_OUT VOUCHER_REDEMPTION"`
	Amount               decimal.Decimal `json:"amount" validate:"required,gt=0"`
	Currency             string          `json:"currency" validate:"required,len=3"`
	Description          *string         `json:"description,omitempty" validate:"omitempty,max=500"`
	RelatedTransactionID *string         `json:"related_transaction_id,omitempty" validate:"omitempty,uuid"`
	SourceAccountID      *string         `json:"source_account_id,omitempty" validate:"omitempty,uuid"`
	DestinationAccountID *string         `json:"destination_account_id,omitempty" validate:"omitempty,uuid"`
	VoucherCode          *string         `json:"voucher_code,omitempty" validate:"omitempty,max=50"`
	ExternalReferenceID  *string         `json:"external_reference_id,omitempty" validate:"omitempty,max=255"`
}

type TransactionUpdateDto struct {
	Status               *TransactionStatus `json:"status,omitempty" validate:"omitempty,oneof=PENDING COMPLETED FAILED CANCELLED"`
	Description          *string            `json:"description,omitempty" validate:"omitempty,max=500"`
	RelatedTransactionID *string            `json:"related_transaction_id,omitempty" validate:"omitempty,uuid"`
	SourceAccountID      *string            `json:"source_account_id,omitempty" validate:"omitempty,uuid"`
	DestinationAccountID *string            `json:"destination_account_id,omitempty" validate:"omitempty,uuid"`
	VoucherCode          *string            `json:"voucher_code,omitempty" validate:"omitempty,max=50"`
	ExternalReferenceID  *string            `json:"external_reference_id,omitempty" validate:"omitempty,max=255"`
	CompletedAt          *time.Time         `json:"completed_at,omitempty"`
}
