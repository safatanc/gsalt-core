package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AccountType represents the type of account
type AccountType string

const (
	AccountTypePersonal AccountType = "PERSONAL"
	AccountTypeMerchant AccountType = "MERCHANT"
)

// AccountStatus represents the status of account
type AccountStatus string

const (
	AccountStatusActive    AccountStatus = "ACTIVE"
	AccountStatusSuspended AccountStatus = "SUSPENDED"
	AccountStatusBlocked   AccountStatus = "BLOCKED"
)

// KYCStatus represents the KYC verification status
type KYCStatus string

const (
	KYCStatusUnverified KYCStatus = "UNVERIFIED"
	KYCStatusPending    KYCStatus = "PENDING"
	KYCStatusVerified   KYCStatus = "VERIFIED"
	KYCStatusRejected   KYCStatus = "REJECTED"
)

type Account struct {
	ConnectID      uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"connect_id"`
	Balance        int64          `json:"balance"`
	Points         int64          `json:"points"`
	AccountType    AccountType    `json:"account_type"`
	Status         AccountStatus  `json:"status"`
	KYCStatus      KYCStatus      `json:"kyc_status"`
	DailyLimit     *int64         `json:"daily_limit"`
	MonthlyLimit   *int64         `json:"monthly_limit"`
	LastActivityAt *time.Time     `json:"last_activity_at"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}
