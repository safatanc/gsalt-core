package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type VoucherType string

const (
	VoucherTypeBalance       VoucherType = "BALANCE"
	VoucherTypeLoyaltyPoints VoucherType = "LOYALTY_POINTS"
	VoucherTypeDiscount      VoucherType = "DISCOUNT"
)

type VoucherStatus string

const (
	VoucherStatusActive   VoucherStatus = "ACTIVE"
	VoucherStatusInactive VoucherStatus = "INACTIVE"
	VoucherStatusRedeemed VoucherStatus = "REDEEMED"
	VoucherStatusExpired  VoucherStatus = "EXPIRED"
)

type Voucher struct {
	ID                 uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code               string           `json:"code"`
	Name               string           `json:"name"`
	Description        *string          `json:"description,omitempty"`
	Type               VoucherType      `json:"type"`
	Value              decimal.Decimal  `gorm:"type:decimal(18,2)" json:"value"`
	Currency           string           `json:"currency"`
	LoyaltyPointsValue *int64           `json:"loyalty_points_value,omitempty"`
	DiscountPercentage *decimal.Decimal `gorm:"type:decimal(5,2)" json:"discount_percentage,omitempty"`
	DiscountAmount     *decimal.Decimal `gorm:"type:decimal(18,2)" json:"discount_amount,omitempty"`
	MaxRedeemCount     int              `json:"max_redeem_count"`
	CurrentRedeemCount int              `json:"current_redeem_count"`
	ValidFrom          time.Time        `json:"valid_from"`
	ValidUntil         *time.Time       `json:"valid_until,omitempty"`
	Status             VoucherStatus    `json:"status"`
	CreatedBy          *uuid.UUID       `json:"created_by,omitempty"`
	CreatedAt          time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt   `gorm:"index" json:"deleted_at"`
}

type VoucherCreateRequest struct {
	Code               string           `json:"code" validate:"required,max=50"`
	Name               string           `json:"name" validate:"required,max=255"`
	Description        *string          `json:"description,omitempty" validate:"omitempty,max=1000"`
	Type               VoucherType      `json:"type" validate:"required,oneof=BALANCE LOYALTY_POINTS DISCOUNT"`
	Value              decimal.Decimal  `json:"value" validate:"required,gt=0"`
	Currency           string           `json:"currency" validate:"required,len=3"`
	LoyaltyPointsValue *int64           `json:"loyalty_points_value,omitempty" validate:"omitempty,min=0"`
	DiscountPercentage *decimal.Decimal `json:"discount_percentage,omitempty" validate:"omitempty,min=0,max=100"`
	DiscountAmount     *decimal.Decimal `json:"discount_amount,omitempty" validate:"omitempty,gt=0"`
	MaxRedeemCount     int              `json:"max_redeem_count" validate:"min=1"`
	ValidFrom          time.Time        `json:"valid_from" validate:"required"`
	ValidUntil         *time.Time       `json:"valid_until,omitempty"`
	CreatedBy          *string          `json:"created_by,omitempty" validate:"omitempty,uuid"`
}

type VoucherUpdateRequest struct {
	Code               *string          `json:"code,omitempty" validate:"omitempty,max=50"`
	Name               *string          `json:"name,omitempty" validate:"omitempty,max=255"`
	Description        *string          `json:"description,omitempty" validate:"omitempty,max=1000"`
	Type               *VoucherType     `json:"type,omitempty" validate:"omitempty,oneof=BALANCE LOYALTY_POINTS DISCOUNT"`
	Value              *decimal.Decimal `json:"value,omitempty" validate:"omitempty,gt=0"`
	Currency           *string          `json:"currency,omitempty" validate:"omitempty,len=3"`
	LoyaltyPointsValue *int64           `json:"loyalty_points_value,omitempty" validate:"omitempty,min=0"`
	DiscountPercentage *decimal.Decimal `json:"discount_percentage,omitempty" validate:"omitempty,min=0,max=100"`
	DiscountAmount     *decimal.Decimal `json:"discount_amount,omitempty" validate:"omitempty,gt=0"`
	MaxRedeemCount     *int             `json:"max_redeem_count,omitempty" validate:"omitempty,min=1"`
	ValidFrom          *time.Time       `json:"valid_from,omitempty"`
	ValidUntil         *time.Time       `json:"valid_until,omitempty"`
	Status             *VoucherStatus   `json:"status,omitempty" validate:"omitempty,oneof=ACTIVE INACTIVE REDEEMED EXPIRED"`
}

type VoucherRedeemRequest struct {
	Code      string `json:"code" validate:"required,max=50"`
	AccountID string `json:"account_id" validate:"required,uuid"`
}
