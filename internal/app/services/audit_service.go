package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"gorm.io/gorm"
)

type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{
		db: db,
	}
}

// LogAudit creates an audit log entry for any change in the system
func (s *AuditService) LogAudit(tableName string, recordID uuid.UUID, action models.AuditAction, oldData, newData interface{}, changedBy *uuid.UUID) error {
	var oldDataJSON, newDataJSON *string

	if oldData != nil {
		jsonBytes, err := json.Marshal(oldData)
		if err != nil {
			return fmt.Errorf("failed to marshal old data: %w", err)
		}
		strJSON := string(jsonBytes)
		oldDataJSON = &strJSON
	}

	if newData != nil {
		jsonBytes, err := json.Marshal(newData)
		if err != nil {
			return fmt.Errorf("failed to marshal new data: %w", err)
		}
		strJSON := string(jsonBytes)
		newDataJSON = &strJSON
	}

	auditLog := &models.AuditLog{
		TableName: tableName,
		RecordID:  recordID,
		Action:    action,
		OldData:   oldDataJSON,
		NewData:   newDataJSON,
		ChangedBy: changedBy,
		ChangedAt: time.Now(),
	}

	if err := s.db.Create(auditLog).Error; err != nil {
		return errors.NewInternalServerError(err, "Failed to create audit log")
	}

	return nil
}

// LogTransactionStatusChange creates a status history entry for a transaction
func (s *AuditService) LogTransactionStatusChange(
	transactionID uuid.UUID,
	fromStatus, toStatus models.TransactionStatus,
	reason string,
	metadata map[string]interface{},
	createdBy *uuid.UUID,
) error {
	var metadataJSON *string
	if metadata != nil {
		jsonBytes, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		strJSON := string(jsonBytes)
		metadataJSON = &strJSON
	}

	history := &models.TransactionStatusHistory{
		TransactionID: transactionID,
		FromStatus:    &fromStatus,
		ToStatus:      toStatus,
		Reason:        &reason,
		Metadata:      metadataJSON,
		CreatedBy:     createdBy,
		CreatedAt:     time.Now(),
	}

	if err := s.db.Create(history).Error; err != nil {
		return errors.NewInternalServerError(err, "Failed to create transaction status history")
	}

	return nil
}

// GetTransactionStatusHistory retrieves the status history for a transaction
func (s *AuditService) GetTransactionStatusHistory(transactionID string) ([]*models.TransactionStatusHistory, error) {
	parsedID, err := uuid.Parse(transactionID)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid transaction ID format")
	}

	var history []*models.TransactionStatusHistory
	if err := s.db.Where("transaction_id = ?", parsedID).
		Order("created_at DESC").
		Find(&history).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to get transaction status history")
	}

	return history, nil
}

// GetAuditLogs retrieves audit logs with pagination
func (s *AuditService) GetAuditLogs(pagination *models.PaginationRequest) (*models.Pagination[[]models.AuditLog], error) {
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}
	if pagination.Page <= 0 {
		pagination.Page = 1
	}

	offset := (pagination.Page - 1) * pagination.Limit

	var totalItems int64
	if err := s.db.Model(&models.AuditLog{}).Count(&totalItems).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to count audit logs")
	}

	var logs []models.AuditLog
	query := s.db.Order("changed_at DESC")

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to get audit logs")
	}

	totalPages := int((totalItems + int64(pagination.Limit) - 1) / int64(pagination.Limit))
	hasNext := pagination.Page < totalPages
	hasPrev := pagination.Page > 1

	result := &models.Pagination[[]models.AuditLog]{
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		TotalPages: totalPages,
		TotalItems: int(totalItems),
		HasNext:    hasNext,
		HasPrev:    hasPrev,
		Items:      logs,
	}

	return result, nil
}
