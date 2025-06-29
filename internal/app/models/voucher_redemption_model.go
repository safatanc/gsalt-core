package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VoucherRedemption struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	VoucherID     uuid.UUID      `json:"voucher_id"`
	AccountID     uuid.UUID      `json:"account_id"`
	TransactionID *uuid.UUID     `json:"transaction_id,omitempty"`
	RedeemedAt    time.Time      `gorm:"autoCreateTime" json:"redeemed_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

type VoucherRedemptionCreateRequest struct {
	VoucherID     string  `json:"voucher_id" validate:"required,uuid"`
	AccountID     string  `json:"account_id" validate:"required,uuid"`
	TransactionID *string `json:"transaction_id,omitempty" validate:"omitempty,uuid"`
}

type VoucherRedemptionUpdateRequest struct {
	TransactionID *string `json:"transaction_id,omitempty" validate:"omitempty,uuid"`
}
