package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Account struct {
	ConnectID uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"connect_id"`
	Balance   int64          `json:"balance"`
	Points    int64          `json:"points"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

type AccountCreateDto struct {
	Balance int64 `json:"balance" validate:"min=0"`
	Points  int64 `json:"points" validate:"min=0"`
}

type AccountUpdateDto struct {
	Balance *int64 `json:"balance,omitempty" validate:"omitempty,min=0"`
	Points  *int64 `json:"points,omitempty" validate:"omitempty,min=0"`
}
