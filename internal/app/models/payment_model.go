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
	ID                   uuid.UUID       `json:"id" db:"id"`
	TransactionID        uuid.UUID       `json:"transaction_id" db:"transaction_id"`
	Provider             string          `json:"provider" db:"provider"`
	ProviderPaymentID    *string         `json:"provider_payment_id" db:"provider_payment_id"`
	PaymentURL           *string         `json:"payment_url" db:"payment_url"`
	QRCode               *string         `json:"qr_code" db:"qr_code"`
	VirtualAccountNumber *string         `json:"virtual_account_number" db:"virtual_account_number"`
	VirtualAccountBank   *string         `json:"virtual_account_bank" db:"virtual_account_bank"`
	RetailOutletCode     *string         `json:"retail_outlet_code" db:"retail_outlet_code"`
	RetailPaymentCode    *string         `json:"retail_payment_code" db:"retail_payment_code"`
	CardToken            *string         `json:"card_token" db:"card_token"`
	ExpiryTime           *time.Time      `json:"expiry_time" db:"expiry_time"`
	PaymentTime          *time.Time      `json:"payment_time" db:"payment_time"`
	ProviderFeeAmount    *int64          `json:"provider_fee_amount" db:"provider_fee_amount"`
	StatusHistory        json.RawMessage `json:"status_history" db:"status_history"`
	RawProviderResponse  json.RawMessage `json:"raw_provider_response" db:"raw_provider_response"`
	CreatedAt            time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at" db:"updated_at"`
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
