package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentStatusPending        PaymentStatus = "PENDING"
	PaymentStatusWaitingPayment PaymentStatus = "WAITING_PAYMENT"
	PaymentStatusProcessing     PaymentStatus = "PROCESSING"
	PaymentStatusCompleted      PaymentStatus = "COMPLETED"
	PaymentStatusFailed         PaymentStatus = "FAILED"
	PaymentStatusExpired        PaymentStatus = "EXPIRED"
	PaymentStatusCancelled      PaymentStatus = "CANCELLED"
)

type PaymentAccountType string

const (
	PaymentAccountTypeBank    PaymentAccountType = "BANK_ACCOUNT"
	PaymentAccountTypeEwallet PaymentAccountType = "EWALLET"
)

type PaymentDetails struct {
	ID                   uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TransactionID        uuid.UUID       `json:"transaction_id" gorm:"type:uuid;not null"`
	Provider             string          `json:"provider" gorm:"type:varchar(50);not null"`
	ProviderPaymentID    *string         `json:"provider_payment_id" gorm:"type:varchar(255)"`
	PaymentURL           *string         `json:"payment_url" gorm:"type:text"`
	QRCode               *string         `json:"qr_code" gorm:"type:text"`
	VirtualAccountNumber *string         `json:"virtual_account_number" gorm:"type:varchar(50)"`
	VirtualAccountBank   *string         `json:"virtual_account_bank" gorm:"type:varchar(10)"`
	RetailOutletCode     *string         `json:"retail_outlet_code" gorm:"type:varchar(10)"`
	RetailPaymentCode    *string         `json:"retail_payment_code" gorm:"type:varchar(50)"`
	CardToken            *string         `json:"card_token" gorm:"type:varchar(255)"`
	ExpiryTime           *time.Time      `json:"expiry_time" gorm:"type:timestamp with time zone"`
	PaymentTime          *time.Time      `json:"payment_time" gorm:"type:timestamp with time zone"`
	ProviderFeeAmount    *int64          `json:"provider_fee_amount" gorm:"type:bigint"`
	StatusHistory        json.RawMessage `json:"status_history" gorm:"type:jsonb"`
	RawProviderResponse  json.RawMessage `json:"raw_provider_response" gorm:"type:jsonb"`
	CreatedAt            time.Time       `json:"created_at" gorm:"type:timestamp with time zone;autoCreateTime"`
	UpdatedAt            time.Time       `json:"updated_at" gorm:"type:timestamp with time zone;autoUpdateTime"`
}

type PaymentAccount struct {
	ID               uuid.UUID          `json:"id" db:"id"`
	AccountID        uuid.UUID          `json:"account_id" db:"account_id"`
	Type             PaymentAccountType `json:"type" db:"type"`
	Provider         string             `json:"provider" db:"provider"`
	AccountNumber    string             `json:"account_number" db:"account_number"`
	AccountName      string             `json:"account_name" db:"account_name"`
	IsVerified       bool               `json:"is_verified" db:"is_verified"`
	VerificationTime *time.Time         `json:"verification_time" db:"verification_time"`
	IsActive         bool               `json:"is_active" db:"is_active"`
	CreatedAt        time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at" db:"updated_at"`
}

// Request/Response structs
type PaymentDetailsCreateRequest struct {
	TransactionID        uuid.UUID       `json:"transaction_id" validate:"required"`
	Provider             string          `json:"provider" validate:"required"`
	ProviderPaymentID    *string         `json:"provider_payment_id"`
	PaymentURL           *string         `json:"payment_url"`
	QRCode               *string         `json:"qr_code"`
	VirtualAccountNumber *string         `json:"virtual_account_number"`
	VirtualAccountBank   *string         `json:"virtual_account_bank"`
	RetailOutletCode     *string         `json:"retail_outlet_code"`
	RetailPaymentCode    *string         `json:"retail_payment_code"`
	CardToken            *string         `json:"card_token"`
	ExpiryTime           *time.Time      `json:"expiry_time"`
	ProviderFeeAmount    *int64          `json:"provider_fee_amount"`
	RawProviderResponse  json.RawMessage `json:"raw_provider_response"`
}

type PaymentAccountCreateRequest struct {
	AccountID     uuid.UUID          `json:"account_id" validate:"required"`
	Type          PaymentAccountType `json:"type" validate:"required,oneof=BANK_ACCOUNT EWALLET"`
	Provider      string             `json:"provider" validate:"required"`
	AccountNumber string             `json:"account_number" validate:"required"`
	AccountName   string             `json:"account_name" validate:"required"`
}

type PaymentStatusUpdateRequest struct {
	Status            PaymentStatus `json:"status" validate:"required,oneof=PENDING WAITING_PAYMENT PROCESSING COMPLETED FAILED EXPIRED CANCELLED"`
	StatusDescription *string       `json:"status_description"`
	ProviderPaymentID *string       `json:"provider_payment_id"`
	PaymentTime       *time.Time    `json:"payment_time"`
}
