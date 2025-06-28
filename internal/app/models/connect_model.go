package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	ConnectUserRoleUser  = "USER"
	ConnectUserRoleAdmin = "ADMIN"
)

type ConnectUser struct {
	ID              uuid.UUID  `json:"id,omitempty"`
	Email           string     `json:"email,omitempty"`
	Username        string     `json:"username,omitempty"`
	FullName        string     `json:"full_name,omitempty"`
	AvatarURL       string     `json:"avatar_url,omitempty"`
	GlobalRole      string     `json:"global_role,omitempty"`
	IsEmailVerified bool       `json:"is_email_verified,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
}
