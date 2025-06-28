package services

import (
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TransactionService struct {
	db             *gorm.DB
	validator      *validator.Validate
	accountService *AccountService
}

func NewTransactionService(db *gorm.DB, validator *validator.Validate, accountService *AccountService) *TransactionService {
	return &TransactionService{
		db:             db,
		validator:      validator,
		accountService: accountService,
	}
}

func (s *TransactionService) CreateTransaction(req *models.TransactionCreateDto) (*models.Transaction, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, errors.NewBadRequestError(err.Error())
	}

	// Parse UUIDs
	accountId, err := uuid.Parse(req.AccountID)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid account ID format")
	}

	// Verify account exists
	_, err = s.accountService.GetAccount(req.AccountID)
	if err != nil {
		return nil, err
	}

	// Create transaction
	transaction := &models.Transaction{
		AccountID:   accountId,
		Type:        req.Type,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Status:      models.TransactionStatusPending,
		Description: req.Description,
	}

	// Parse optional UUIDs
	if req.RelatedTransactionID != nil {
		relatedId, err := uuid.Parse(*req.RelatedTransactionID)
		if err != nil {
			return nil, errors.NewBadRequestError("Invalid related transaction ID format")
		}
		transaction.RelatedTransactionID = &relatedId
	}

	if req.SourceAccountID != nil {
		sourceId, err := uuid.Parse(*req.SourceAccountID)
		if err != nil {
			return nil, errors.NewBadRequestError("Invalid source account ID format")
		}
		transaction.SourceAccountID = &sourceId
	}

	if req.DestinationAccountID != nil {
		destId, err := uuid.Parse(*req.DestinationAccountID)
		if err != nil {
			return nil, errors.NewBadRequestError("Invalid destination account ID format")
		}
		transaction.DestinationAccountID = &destId
	}

	transaction.VoucherCode = req.VoucherCode
	transaction.ExternalReferenceID = req.ExternalReferenceID

	if err := s.db.Create(transaction).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to create transaction")
	}

	return transaction, nil
}

func (s *TransactionService) GetTransaction(transactionId string) (*models.Transaction, error) {
	transactionUUID, err := uuid.Parse(transactionId)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid transaction ID format")
	}

	var transaction models.Transaction
	err = s.db.Where("id = ?", transactionUUID).First(&transaction).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Transaction not found")
		}
		return nil, errors.NewInternalServerError(err, "Failed to get transaction")
	}

	return &transaction, nil
}

func (s *TransactionService) GetTransactionsByAccount(accountId string, limit, offset int) ([]models.Transaction, error) {
	accountUUID, err := uuid.Parse(accountId)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid account ID format")
	}

	var transactions []models.Transaction
	query := s.db.Where("account_id = ?", accountUUID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err = query.Find(&transactions).Error
	if err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to get transactions")
	}

	return transactions, nil
}

func (s *TransactionService) UpdateTransaction(transactionId string, req *models.TransactionUpdateDto) (*models.Transaction, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, errors.NewBadRequestError(err.Error())
	}

	transaction, err := s.GetTransaction(transactionId)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Status != nil {
		transaction.Status = *req.Status
	}
	if req.Description != nil {
		transaction.Description = req.Description
	}
	if req.RelatedTransactionID != nil {
		relatedId, err := uuid.Parse(*req.RelatedTransactionID)
		if err != nil {
			return nil, errors.NewBadRequestError("Invalid related transaction ID format")
		}
		transaction.RelatedTransactionID = &relatedId
	}
	if req.SourceAccountID != nil {
		sourceId, err := uuid.Parse(*req.SourceAccountID)
		if err != nil {
			return nil, errors.NewBadRequestError("Invalid source account ID format")
		}
		transaction.SourceAccountID = &sourceId
	}
	if req.DestinationAccountID != nil {
		destId, err := uuid.Parse(*req.DestinationAccountID)
		if err != nil {
			return nil, errors.NewBadRequestError("Invalid destination account ID format")
		}
		transaction.DestinationAccountID = &destId
	}
	if req.VoucherCode != nil {
		transaction.VoucherCode = req.VoucherCode
	}
	if req.ExternalReferenceID != nil {
		transaction.ExternalReferenceID = req.ExternalReferenceID
	}
	if req.CompletedAt != nil {
		transaction.CompletedAt = req.CompletedAt
	}

	if err := s.db.Save(transaction).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to update transaction")
	}

	return transaction, nil
}

func (s *TransactionService) ProcessTopup(accountId string, amount decimal.Decimal, externalRefId *string) (*models.Transaction, error) {
	// Verify account exists
	account, err := s.accountService.GetAccount(accountId)
	if err != nil {
		return nil, err
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create topup transaction
	transaction := &models.Transaction{
		AccountID:           account.ConnectID,
		Type:                models.TransactionTypeTopup,
		Amount:              amount,
		Currency:            "IDR",
		Status:              models.TransactionStatusCompleted,
		Description:         stringPtr("Balance topup"),
		ExternalReferenceID: externalRefId,
	}

	if err := tx.Create(transaction).Error; err != nil {
		tx.Rollback()
		return nil, errors.NewInternalServerError(err, "Failed to create topup transaction")
	}

	// Update account balance
	account.Balance += amount.IntPart()
	if err := tx.Save(account).Error; err != nil {
		tx.Rollback()
		return nil, errors.NewInternalServerError(err, "Failed to update account balance")
	}

	tx.Commit()
	return transaction, nil
}

func (s *TransactionService) ProcessTransfer(sourceAccountId, destAccountId string, amount decimal.Decimal, description *string) (*models.Transaction, *models.Transaction, error) {
	// Verify accounts exist
	sourceAccount, err := s.accountService.GetAccount(sourceAccountId)
	if err != nil {
		return nil, nil, err
	}

	destAccount, err := s.accountService.GetAccount(destAccountId)
	if err != nil {
		return nil, nil, err
	}

	// Check sufficient balance
	if sourceAccount.Balance < amount.IntPart() {
		return nil, nil, errors.NewBadRequestError("Insufficient balance")
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create transfer out transaction
	transferOut := &models.Transaction{
		AccountID:            sourceAccount.ConnectID,
		Type:                 models.TransactionTypeTransferOut,
		Amount:               amount,
		Currency:             "IDR",
		Status:               models.TransactionStatusCompleted,
		Description:          description,
		DestinationAccountID: &destAccount.ConnectID,
	}

	if err := tx.Create(transferOut).Error; err != nil {
		tx.Rollback()
		return nil, nil, errors.NewInternalServerError(err, "Failed to create transfer out transaction")
	}

	// Create transfer in transaction
	transferIn := &models.Transaction{
		AccountID:            destAccount.ConnectID,
		Type:                 models.TransactionTypeTransferIn,
		Amount:               amount,
		Currency:             "IDR",
		Status:               models.TransactionStatusCompleted,
		Description:          description,
		SourceAccountID:      &sourceAccount.ConnectID,
		RelatedTransactionID: &transferOut.ID,
	}

	if err := tx.Create(transferIn).Error; err != nil {
		tx.Rollback()
		return nil, nil, errors.NewInternalServerError(err, "Failed to create transfer in transaction")
	}

	// Update related transaction ID
	transferOut.RelatedTransactionID = &transferIn.ID
	if err := tx.Save(transferOut).Error; err != nil {
		tx.Rollback()
		return nil, nil, errors.NewInternalServerError(err, "Failed to update transfer out transaction")
	}

	// Update account balances
	sourceAccount.Balance -= amount.IntPart()
	destAccount.Balance += amount.IntPart()

	if err := tx.Save(sourceAccount).Error; err != nil {
		tx.Rollback()
		return nil, nil, errors.NewInternalServerError(err, "Failed to update source account balance")
	}

	if err := tx.Save(destAccount).Error; err != nil {
		tx.Rollback()
		return nil, nil, errors.NewInternalServerError(err, "Failed to update destination account balance")
	}

	tx.Commit()
	return transferOut, transferIn, nil
}

func (s *TransactionService) ProcessPayment(accountId string, amount decimal.Decimal, description *string, externalRefId *string) (*models.Transaction, error) {
	// Verify account exists
	account, err := s.accountService.GetAccount(accountId)
	if err != nil {
		return nil, err
	}

	// Check sufficient balance
	if account.Balance < amount.IntPart() {
		return nil, errors.NewBadRequestError("Insufficient balance")
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create payment transaction
	transaction := &models.Transaction{
		AccountID:           account.ConnectID,
		Type:                models.TransactionTypePayment,
		Amount:              amount,
		Currency:            "IDR",
		Status:              models.TransactionStatusCompleted,
		Description:         description,
		ExternalReferenceID: externalRefId,
	}

	if err := tx.Create(transaction).Error; err != nil {
		tx.Rollback()
		return nil, errors.NewInternalServerError(err, "Failed to create payment transaction")
	}

	// Update account balance
	account.Balance -= amount.IntPart()
	if err := tx.Save(account).Error; err != nil {
		tx.Rollback()
		return nil, errors.NewInternalServerError(err, "Failed to update account balance")
	}

	tx.Commit()
	return transaction, nil
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
