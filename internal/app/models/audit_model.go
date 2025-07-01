package models

import (
	"time"

	"github.com/google/uuid"
)

// AuditAction represents the type of action being audited
type AuditAction string

const (
	AuditActionCreate       AuditAction = "CREATE"
	AuditActionUpdate       AuditAction = "UPDATE"
	AuditActionDelete       AuditAction = "DELETE"
	AuditActionStatusChange AuditAction = "STATUS_CHANGE"
)

// AuditLog represents a record of changes made to any entity in the system
type AuditLog struct {
	ID        uuid.UUID   `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TableName string      `json:"table_name" gorm:"type:varchar(50);not null"`
	RecordID  uuid.UUID   `json:"record_id" gorm:"type:uuid;not null"`
	Action    AuditAction `json:"action" gorm:"type:audit_action;not null"`
	OldData   *string     `json:"old_data" gorm:"type:jsonb"`
	NewData   *string     `json:"new_data" gorm:"type:jsonb"`
	ChangedBy *uuid.UUID  `json:"changed_by" gorm:"type:uuid"`
	ChangedAt time.Time   `json:"changed_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// TransactionStatusHistory represents the history of status changes for a transaction
type TransactionStatusHistory struct {
	ID            uuid.UUID          `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TransactionID uuid.UUID          `json:"transaction_id" gorm:"type:uuid;not null"`
	FromStatus    *TransactionStatus `json:"from_status" gorm:"type:transaction_status"`
	ToStatus      TransactionStatus  `json:"to_status" gorm:"type:transaction_status;not null"`
	Reason        *string            `json:"reason" gorm:"type:text"`
	Metadata      *string            `json:"metadata" gorm:"type:jsonb"`
	CreatedBy     *uuid.UUID         `json:"created_by" gorm:"type:uuid"`
	CreatedAt     time.Time          `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
}
