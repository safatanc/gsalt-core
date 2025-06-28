package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/infrastructures"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Xendit payment method constants for compatibility
const (
	XenditPaymentMethodQRIS           = "QRIS"
	XenditPaymentMethodVirtualAccount = "VIRTUAL_ACCOUNT"
	XenditPaymentMethodEWallet        = "EWALLET"
)

type TransactionService struct {
	db             *gorm.DB
	validator      *infrastructures.Validator
	accountService *AccountService
	xenditService  *XenditService
}

func NewTransactionService(db *gorm.DB, validator *infrastructures.Validator, accountService *AccountService, xenditService *XenditService) *TransactionService {
	return &TransactionService{
		db:             db,
		validator:      validator,
		accountService: accountService,
		xenditService:  xenditService,
	}
}

// Transaction limits configuration (in GSALT units)
type TransactionLimits struct {
	MinTopupAmount     int64 // In GSALT units (100 units = 1 GSALT)
	MaxTopupAmount     int64
	MinTransferAmount  int64
	MaxTransferAmount  int64
	MinPaymentAmount   int64
	MaxPaymentAmount   int64
	DailyTransferLimit int64
	DailyPaymentLimit  int64
}

// Default transaction limits (in GSALT units)
var defaultLimits = TransactionLimits{
	MinTopupAmount:     1000,     // 10 GSALT (10,000 IDR)
	MaxTopupAmount:     5000000,  // 50,000 GSALT (50,000,000 IDR)
	MinTransferAmount:  100,      // 1 GSALT (1,000 IDR)
	MaxTransferAmount:  2500000,  // 25,000 GSALT (25,000,000 IDR)
	MinPaymentAmount:   100,      // 1 GSALT (1,000 IDR)
	MaxPaymentAmount:   1000000,  // 10,000 GSALT (10,000,000 IDR)
	DailyTransferLimit: 10000000, // 100,000 GSALT (100,000,000 IDR)
	DailyPaymentLimit:  5000000,  // 50,000 GSALT (50,000,000 IDR)
}

// Helper function to compare balance (both are in GSALT units)
func (s *TransactionService) hasSufficientBalance(balance int64, amountGsaltUnits int64) bool {
	return balance >= amountGsaltUnits
}

// Helper function to check idempotency
func (s *TransactionService) checkIdempotency(externalRefId *string) (*models.Transaction, error) {
	if externalRefId == nil || *externalRefId == "" {
		return nil, nil // No external reference, proceed normally
	}

	var existingTransaction models.Transaction
	err := s.db.Where("external_reference_id = ?", *externalRefId).First(&existingTransaction).Error
	if err == nil {
		// Transaction with same external reference already exists
		return &existingTransaction, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, errors.NewInternalServerError(err, "Failed to check idempotency")
	}
	return nil, nil // No existing transaction, proceed
}

func (s *TransactionService) CreateTransaction(req *models.TransactionCreateDto) (*models.Transaction, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, errors.NewBadRequestError(err.Error())
	}

	// Check idempotency
	if existingTxn, err := s.checkIdempotency(req.ExternalReferenceID); err != nil {
		return nil, err
	} else if existingTxn != nil {
		return existingTxn, nil // Return existing transaction
	}

	// Parse UUIDs
	accountId, err := uuid.Parse(req.AccountID)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid account ID format")
	}

	var transaction *models.Transaction

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Lock and verify account with SELECT FOR UPDATE
		var account models.Account
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("connect_id = ?", accountId).First(&account).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.NewNotFoundError("Account not found")
			}
			return errors.NewInternalServerError(err, "Failed to get account")
		}

		// Create transaction
		transaction = &models.Transaction{
			AccountID:        accountId,
			Type:             req.Type,
			AmountGsaltUnits: req.AmountGsaltUnits,
			Currency:         req.Currency,
			Status:           models.TransactionStatusPending,
			Description:      req.Description,
		}

		// Set payment fields if provided
		if req.ExchangeRateIDR != nil {
			transaction.ExchangeRateIDR = req.ExchangeRateIDR
		}
		if req.PaymentAmount != nil {
			transaction.PaymentAmount = req.PaymentAmount
		}
		if req.PaymentCurrency != nil {
			transaction.PaymentCurrency = req.PaymentCurrency
		}
		if req.PaymentMethod != nil {
			transaction.PaymentMethod = req.PaymentMethod
		}

		// Parse optional UUIDs
		if req.RelatedTransactionID != nil {
			relatedId, err := uuid.Parse(*req.RelatedTransactionID)
			if err != nil {
				return errors.NewBadRequestError("Invalid related transaction ID format")
			}
			transaction.RelatedTransactionID = &relatedId
		}

		if req.SourceAccountID != nil {
			sourceId, err := uuid.Parse(*req.SourceAccountID)
			if err != nil {
				return errors.NewBadRequestError("Invalid source account ID format")
			}

			// Lock and verify source account
			var sourceAccount models.Account
			if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("connect_id = ?", sourceId).First(&sourceAccount).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return errors.NewNotFoundError("Source account not found")
				}
				return errors.NewInternalServerError(err, "Failed to get source account")
			}

			// Check sufficient balance
			if !s.hasSufficientBalance(sourceAccount.Balance, req.AmountGsaltUnits) {
				return errors.NewBadRequestError("Insufficient balance [" + ErrCodeInsufficientBalance + "]")
			}

			transaction.SourceAccountID = &sourceId
		}

		if req.DestinationAccountID != nil {
			destId, err := uuid.Parse(*req.DestinationAccountID)
			if err != nil {
				return errors.NewBadRequestError("Invalid destination account ID format")
			}
			transaction.DestinationAccountID = &destId
		}

		transaction.VoucherCode = req.VoucherCode
		transaction.ExternalReferenceID = req.ExternalReferenceID

		if err := tx.Create(transaction).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create transaction")
		}

		return nil
	})

	if err != nil {
		return nil, err
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

func (s *TransactionService) GetTransactionsByAccount(accountId string, pagination *models.PaginationRequest) (*models.Pagination[[]models.Transaction], error) {
	accountUUID, err := uuid.Parse(accountId)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid account ID format")
	}

	// Set defaults
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}
	if pagination.Page <= 0 {
		pagination.Page = 1
	}

	offset := (pagination.Page - 1) * pagination.Limit

	// Count total items
	var totalItems int64
	if err := s.db.Model(&models.Transaction{}).Where("account_id = ?", accountUUID).Count(&totalItems).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to count transactions")
	}

	var transactions []models.Transaction
	query := s.db.Where("account_id = ?", accountUUID).Order("created_at DESC")

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err = query.Find(&transactions).Error
	if err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to get transactions")
	}

	// Calculate pagination metadata
	totalPages := int((totalItems + int64(pagination.Limit) - 1) / int64(pagination.Limit))
	hasNext := pagination.Page < totalPages
	hasPrev := pagination.Page > 1

	result := &models.Pagination[[]models.Transaction]{
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		TotalPages: totalPages,
		TotalItems: int(totalItems),
		HasNext:    hasNext,
		HasPrev:    hasPrev,
		Items:      transactions,
	}

	return result, nil
}

func (s *TransactionService) UpdateTransaction(transactionId string, req *models.TransactionUpdateDto) (*models.Transaction, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, errors.NewBadRequestError(err.Error())
	}

	transaction, err := s.GetTransaction(transactionId)
	if err != nil {
		return nil, err
	}

	// Validate status transition if status is being updated
	if req.Status != nil {
		if err := s.validateStatusTransition(transaction.Status, *req.Status); err != nil {
			return nil, err
		}
	}

	// Update fields if provided
	if req.Status != nil {
		transaction.Status = *req.Status
		// Auto-set CompletedAt when status changes to COMPLETED
		if *req.Status == models.TransactionStatusCompleted && transaction.CompletedAt == nil {
			now := time.Now()
			transaction.CompletedAt = &now
		}
	}
	if req.ExchangeRateIDR != nil {
		transaction.ExchangeRateIDR = req.ExchangeRateIDR
	}
	if req.PaymentAmount != nil {
		transaction.PaymentAmount = req.PaymentAmount
	}
	if req.PaymentCurrency != nil {
		transaction.PaymentCurrency = req.PaymentCurrency
	}
	if req.PaymentMethod != nil {
		transaction.PaymentMethod = req.PaymentMethod
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

func (s *TransactionService) ProcessTopup(accountId string, amountGsaltUnits int64, paymentAmount *int64, paymentCurrency *string, paymentMethod *string, externalRefId *string) (*models.TopupResponse, error) {
	// Parse UUID first
	accountUUID, err := uuid.Parse(accountId)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid account ID format")
	}

	// Validate transaction amount
	if err := s.validateTransactionAmount(models.TransactionTypeTopup, amountGsaltUnits); err != nil {
		return nil, err
	}

	// Check idempotency
	if existingTxn, err := s.checkIdempotency(externalRefId); err != nil {
		return nil, err
	} else if existingTxn != nil {
		// Create response for existing transaction
		paymentInstructions, _ := existingTxn.GetPaymentInstructionsAsMap()
		return &models.TopupResponse{
			Transaction:         existingTxn,
			PaymentInstructions: paymentInstructions,
		}, nil
	}

	// Check daily limits
	if err := s.checkDailyLimits(accountUUID, models.TransactionTypeTopup, amountGsaltUnits); err != nil {
		return nil, err
	}

	// Auto-calculate payment_amount if not provided but payment_currency is provided
	if paymentAmount == nil && paymentCurrency != nil {
		// Convert GSALT units to IDR (1 GSALT = 100 units = 1000 IDR)
		gsaltAmount := decimal.NewFromInt(amountGsaltUnits).Div(decimal.NewFromInt(100))
		idrAmount := gsaltAmount.Mul(decimal.NewFromInt(1000))
		calculatedAmount := idrAmount.IntPart()
		paymentAmount = &calculatedAmount
	}

	var transaction *models.Transaction
	now := time.Now()
	exchangeRate := decimal.NewFromInt(1000) // 1 GSALT = 1000 IDR

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Determine if this is an external payment method
		isExternalPayment := paymentMethod != nil && (*paymentMethod == "QRIS" || *paymentMethod == "BANK_TRANSFER" || *paymentMethod == "EWALLET")

		// Set initial status
		status := models.TransactionStatusCompleted
		var completedAt *time.Time = &now

		if isExternalPayment {
			status = models.TransactionStatusPending
			completedAt = nil
		}

		// Create transaction record
		description := fmt.Sprintf("Topup %d GSALT", amountGsaltUnits/100)
		transaction = &models.Transaction{
			AccountID:        accountUUID,
			Type:             models.TransactionTypeTopup,
			AmountGsaltUnits: amountGsaltUnits,
			Currency:         "GSALT",
			ExchangeRateIDR:  &exchangeRate,
			Status:           status,
			Description:      &description,
			CompletedAt:      completedAt,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		// Set payment fields if provided
		if paymentAmount != nil {
			transaction.PaymentAmount = paymentAmount
		}
		if paymentCurrency != nil {
			transaction.PaymentCurrency = paymentCurrency
		}
		if paymentMethod != nil {
			transaction.PaymentMethod = paymentMethod
		}
		if externalRefId != nil {
			transaction.ExternalReferenceID = externalRefId
		}

		// Create transaction in database
		if err := tx.Create(transaction).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create topup transaction")
		}

		// For external payments, create Xendit payment request
		if isExternalPayment && s.xenditService != nil {
			var xenditPaymentMethod models.PaymentMethod = models.PaymentMethodQRIS // Default to QRIS
			if paymentMethod != nil {
				switch *paymentMethod {
				case "QRIS":
					xenditPaymentMethod = models.PaymentMethodQRIS
				case "BANK_TRANSFER":
					xenditPaymentMethod = models.PaymentMethodVirtualAccount
				case "EWALLET":
					xenditPaymentMethod = models.PaymentMethodEWallet
				}
			}

			// Get account info for customer details
			var account models.Account
			if err := tx.Where("id = ?", accountUUID).First(&account).Error; err != nil {
				return errors.NewInternalServerError(err, "Failed to get account details")
			}

			// Create payment request with Xendit
			paymentResp, err := s.xenditService.CreateTopupPayment(
				context.Background(),
				transaction.ID.String(),
				*paymentAmount,
				xenditPaymentMethod,
				&account,
			)
			if err != nil {
				return errors.NewInternalServerError(err, "Failed to create payment request with Xendit")
			}

			// Convert payment response to JSON for storage
			paymentInstructionsJSON, err := s.xenditService.ConvertPaymentResponseToJSON(paymentResp)
			if err != nil {
				return errors.NewInternalServerError(err, "Failed to convert payment instructions to JSON")
			}

			// Update transaction with Xendit payment request ID and payment instructions
			transaction.ExternalReferenceID = &paymentResp.PaymentRequestID
			transaction.PaymentInstructions = &paymentInstructionsJSON
			if err := tx.Save(transaction).Error; err != nil {
				return errors.NewInternalServerError(err, "Failed to update transaction with Xendit payment data")
			}
		}

		// For non-external payments (direct balance credit), update account balance
		if !isExternalPayment {
			if err := s.updateAccountBalance(tx, accountUUID, amountGsaltUnits); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Create response with payment instructions
	paymentInstructions, _ := transaction.GetPaymentInstructionsAsMap()
	response := &models.TopupResponse{
		Transaction:         transaction,
		PaymentInstructions: paymentInstructions,
	}

	return response, nil
}

// updateAccountBalance updates the account balance with the given amount
func (s *TransactionService) updateAccountBalance(tx *gorm.DB, accountID uuid.UUID, amountGsaltUnits int64) error {
	// Lock and get account
	var account models.Account
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", accountID).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewNotFoundError("Account not found")
		}
		return errors.NewInternalServerError(err, "Failed to get account")
	}

	// Update balance
	account.Balance += amountGsaltUnits
	if err := tx.Save(&account).Error; err != nil {
		return errors.NewInternalServerError(err, "Failed to update account balance")
	}

	return nil
}

func (s *TransactionService) ProcessTransfer(sourceAccountId, destAccountId string, amountGsaltUnits int64, description *string) (*models.Transaction, *models.Transaction, error) {
	// Parse UUIDs first
	sourceUUID, err := uuid.Parse(sourceAccountId)
	if err != nil {
		return nil, nil, errors.NewBadRequestError("Invalid source account ID format")
	}

	destUUID, err := uuid.Parse(destAccountId)
	if err != nil {
		return nil, nil, errors.NewBadRequestError("Invalid destination account ID format")
	}

	// Check if trying to transfer to same account
	if sourceUUID == destUUID {
		return nil, nil, errors.NewBadRequestError("Cannot transfer to the same account [" + ErrCodeSelfTransfer + "]")
	}

	// Validate transaction amount
	if err := s.validateTransactionAmount(models.TransactionTypeTransferOut, amountGsaltUnits); err != nil {
		return nil, nil, err
	}

	// Check daily limits
	if err := s.checkDailyLimits(sourceUUID, models.TransactionTypeTransferOut, amountGsaltUnits); err != nil {
		return nil, nil, err
	}

	var transferOut, transferIn *models.Transaction
	now := time.Now()

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Lock and verify source account with SELECT FOR UPDATE
		var sourceAccount models.Account
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("connect_id = ?", sourceUUID).First(&sourceAccount).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.NewNotFoundError("Source account not found")
			}
			return errors.NewInternalServerError(err, "Failed to get source account")
		}

		// Lock and verify destination account with SELECT FOR UPDATE
		var destAccount models.Account
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("connect_id = ?", destUUID).First(&destAccount).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.NewNotFoundError("Destination account not found")
			}
			return errors.NewInternalServerError(err, "Failed to get destination account")
		}

		// Check sufficient balance
		if !s.hasSufficientBalance(sourceAccount.Balance, amountGsaltUnits) {
			return errors.NewBadRequestError("Insufficient balance [" + ErrCodeInsufficientBalance + "]")
		}

		// Create transfer out transaction
		transferOut = &models.Transaction{
			AccountID:            sourceAccount.ConnectID,
			Type:                 models.TransactionTypeTransferOut,
			AmountGsaltUnits:     amountGsaltUnits,
			Currency:             "GSALT",
			Status:               models.TransactionStatusCompleted,
			Description:          description,
			DestinationAccountID: &destAccount.ConnectID,
			CompletedAt:          &now,
		}

		if err := tx.Create(transferOut).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create transfer out transaction")
		}

		// Create transfer in transaction
		transferIn = &models.Transaction{
			AccountID:            destAccount.ConnectID,
			Type:                 models.TransactionTypeTransferIn,
			AmountGsaltUnits:     amountGsaltUnits,
			Currency:             "GSALT",
			Status:               models.TransactionStatusCompleted,
			Description:          description,
			SourceAccountID:      &sourceAccount.ConnectID,
			RelatedTransactionID: &transferOut.ID,
			CompletedAt:          &now,
		}

		if err := tx.Create(transferIn).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create transfer in transaction")
		}

		// Update related transaction ID
		transferOut.RelatedTransactionID = &transferIn.ID
		if err := tx.Save(transferOut).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to update transfer out transaction")
		}

		// Update account balances atomically (all in GSALT units)
		sourceAccount.Balance -= amountGsaltUnits
		destAccount.Balance += amountGsaltUnits

		if err := tx.Save(&sourceAccount).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to update source account balance")
		}

		if err := tx.Save(&destAccount).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to update destination account balance")
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return transferOut, transferIn, nil
}

func (s *TransactionService) ProcessPayment(accountId string, amountGsaltUnits int64, paymentAmount *int64, paymentCurrency *string, paymentMethod *string, description *string, externalRefId *string) (*models.Transaction, error) {
	// Parse UUID first
	accountUUID, err := uuid.Parse(accountId)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid account ID format")
	}

	// Validate transaction amount
	if err := s.validateTransactionAmount(models.TransactionTypePayment, amountGsaltUnits); err != nil {
		return nil, err
	}

	// Check daily limits
	if err := s.checkDailyLimits(accountUUID, models.TransactionTypePayment, amountGsaltUnits); err != nil {
		return nil, err
	}

	// Check idempotency
	if existingTxn, err := s.checkIdempotency(externalRefId); err != nil {
		return nil, err
	} else if existingTxn != nil {
		return existingTxn, nil // Return existing transaction
	}

	var transaction *models.Transaction
	now := time.Now()
	exchangeRate := decimal.NewFromFloat(1000.00) // Default: 1000 IDR = 1 GSALT

	// Auto-calculate payment_amount if payment_currency is provided but payment_amount is not
	// This ensures compliance with the database constraint chk_payment_fields_consistency
	if paymentCurrency != nil && paymentAmount == nil {
		// Calculate payment amount based on GSALT amount and exchange rate
		gsaltDecimal := decimal.NewFromInt(amountGsaltUnits).Div(decimal.NewFromInt(100)) // Convert units to GSALT

		// Calculate payment amount based on currency
		switch *paymentCurrency {
		case "IDR":
			calculatedAmount := gsaltDecimal.Mul(exchangeRate).IntPart()
			paymentAmount = &calculatedAmount
		case "GSALT":
			calculatedAmount := amountGsaltUnits
			paymentAmount = &calculatedAmount
		default:
			// For other currencies (USD, EUR, SGD), use IDR as default for now
			// TODO: Implement proper exchange rates for other currencies
			calculatedAmount := gsaltDecimal.Mul(exchangeRate).IntPart()
			paymentAmount = &calculatedAmount
		}
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Lock and verify account with SELECT FOR UPDATE
		var account models.Account
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("connect_id = ?", accountUUID).First(&account).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.NewNotFoundError("Account not found")
			}
			return errors.NewInternalServerError(err, "Failed to get account")
		}

		// For GSALT_BALANCE payment method, check sufficient balance
		if paymentMethod != nil && *paymentMethod == "GSALT_BALANCE" {
			if !s.hasSufficientBalance(account.Balance, amountGsaltUnits) {
				return errors.NewBadRequestError("Insufficient balance [" + ErrCodeInsufficientBalance + "]")
			}
		}

		// Create payment transaction
		transaction = &models.Transaction{
			AccountID:           account.ConnectID,
			Type:                models.TransactionTypePayment,
			AmountGsaltUnits:    amountGsaltUnits,
			Currency:            "GSALT",
			ExchangeRateIDR:     &exchangeRate,
			PaymentAmount:       paymentAmount,
			PaymentCurrency:     paymentCurrency,
			PaymentMethod:       paymentMethod,
			Status:              models.TransactionStatusCompleted,
			Description:         description,
			ExternalReferenceID: externalRefId,
			CompletedAt:         &now,
		}

		if err := tx.Create(transaction).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create payment transaction")
		}

		// Update account balance only if paying with GSALT_BALANCE
		if paymentMethod != nil && *paymentMethod == "GSALT_BALANCE" {
			account.Balance -= amountGsaltUnits
			if err := tx.Save(&account).Error; err != nil {
				return errors.NewInternalServerError(err, "Failed to update account balance")
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return transaction, nil
}

// Helper function to validate transaction amounts (in GSALT units)
func (s *TransactionService) validateTransactionAmount(txnType models.TransactionType, amountGsaltUnits int64) error {
	switch txnType {
	case models.TransactionTypeTopup:
		if amountGsaltUnits < defaultLimits.MinTopupAmount {
			return errors.NewBadRequestError("Topup amount below minimum limit [" + ErrCodeAmountBelowMinimum + "]")
		}
		if amountGsaltUnits > defaultLimits.MaxTopupAmount {
			return errors.NewBadRequestError("Topup amount exceeds maximum limit [" + ErrCodeAmountAboveMaximum + "]")
		}
	case models.TransactionTypeTransferOut:
		if amountGsaltUnits < defaultLimits.MinTransferAmount {
			return errors.NewBadRequestError("Transfer amount below minimum limit [" + ErrCodeAmountBelowMinimum + "]")
		}
		if amountGsaltUnits > defaultLimits.MaxTransferAmount {
			return errors.NewBadRequestError("Transfer amount exceeds maximum limit [" + ErrCodeAmountAboveMaximum + "]")
		}
	case models.TransactionTypePayment:
		if amountGsaltUnits < defaultLimits.MinPaymentAmount {
			return errors.NewBadRequestError("Payment amount below minimum limit [" + ErrCodeAmountBelowMinimum + "]")
		}
		if amountGsaltUnits > defaultLimits.MaxPaymentAmount {
			return errors.NewBadRequestError("Payment amount exceeds maximum limit [" + ErrCodeAmountAboveMaximum + "]")
		}
	}
	return nil
}

// Helper function to check daily limits (in GSALT units)
func (s *TransactionService) checkDailyLimits(accountId uuid.UUID, txnType models.TransactionType, amountGsaltUnits int64) error {
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	var totalToday int64
	var dailyLimit int64

	switch txnType {
	case models.TransactionTypeTransferOut:
		dailyLimit = defaultLimits.DailyTransferLimit
		s.db.Model(&models.Transaction{}).
			Where("account_id = ? AND type = ? AND created_at >= ? AND created_at < ? AND status = ?",
				accountId, models.TransactionTypeTransferOut, today, tomorrow, models.TransactionStatusCompleted).
			Select("COALESCE(SUM(amount_gsalt_units), 0)").Scan(&totalToday)
	case models.TransactionTypePayment:
		dailyLimit = defaultLimits.DailyPaymentLimit
		s.db.Model(&models.Transaction{}).
			Where("account_id = ? AND type = ? AND created_at >= ? AND created_at < ? AND status = ?",
				accountId, models.TransactionTypePayment, today, tomorrow, models.TransactionStatusCompleted).
			Select("COALESCE(SUM(amount_gsalt_units), 0)").Scan(&totalToday)
	default:
		return nil // No daily limit for other transaction types
	}

	if totalToday+amountGsaltUnits > dailyLimit {
		return errors.NewBadRequestError("Daily transaction limit exceeded [" + ErrCodeDailyLimitExceeded + "]")
	}

	return nil
}

// Helper function to validate state transitions
func (s *TransactionService) validateStatusTransition(currentStatus, newStatus models.TransactionStatus) error {
	validTransitions := map[models.TransactionStatus][]models.TransactionStatus{
		models.TransactionStatusPending: {
			models.TransactionStatusCompleted,
			models.TransactionStatusFailed,
			models.TransactionStatusCancelled,
		},
		models.TransactionStatusCompleted: {}, // No transitions allowed from completed
		models.TransactionStatusFailed: {
			models.TransactionStatusPending, // Allow retry
		},
		models.TransactionStatusCancelled: {}, // No transitions allowed from cancelled
	}

	allowedStatuses, exists := validTransitions[currentStatus]
	if !exists {
		return errors.NewBadRequestError("Invalid current transaction status [" + ErrCodeInvalidStatusTransition + "]")
	}

	for _, allowed := range allowedStatuses {
		if allowed == newStatus {
			return nil
		}
	}

	return errors.NewBadRequestError("Invalid status transition [" + ErrCodeInvalidStatusTransition + "]")
}

// Method to expire old pending transactions
func (s *TransactionService) ExpirePendingTransactions() error {
	expiryTime := time.Now().Add(-24 * time.Hour) // 24 hours ago

	result := s.db.Model(&models.Transaction{}).
		Where("status = ? AND created_at < ?", models.TransactionStatusPending, expiryTime).
		Updates(map[string]interface{}{
			"status":     models.TransactionStatusCancelled,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return errors.NewInternalServerError(result.Error, "Failed to expire pending transactions")
	}

	return nil
}

// Error codes for better client error handling
const (
	ErrCodeInsufficientBalance     = "INSUFFICIENT_BALANCE"
	ErrCodeDailyLimitExceeded      = "DAILY_LIMIT_EXCEEDED"
	ErrCodeAmountBelowMinimum      = "AMOUNT_BELOW_MINIMUM"
	ErrCodeAmountAboveMaximum      = "AMOUNT_ABOVE_MAXIMUM"
	ErrCodeSelfTransfer            = "SELF_TRANSFER_NOT_ALLOWED"
	ErrCodeInvalidStatusTransition = "INVALID_STATUS_TRANSITION"
	ErrCodeDuplicateTransaction    = "DUPLICATE_TRANSACTION"
	ErrCodeAccountNotFound         = "ACCOUNT_NOT_FOUND"
	ErrCodeTransactionNotFound     = "TRANSACTION_NOT_FOUND"
)

// Supported currencies for payments
var supportedCurrencies = map[string]bool{
	"GSALT": true,
	"IDR":   true,
	"USD":   true,
	"EUR":   true,
	"SGD":   true,
}

// Helper function to validate currency
func (s *TransactionService) validateCurrency(currency string) error {
	if !supportedCurrencies[currency] {
		return errors.NewBadRequestError("Unsupported currency: " + currency)
	}
	return nil
}

// ProcessGiftIn - Give gift/reward to account (in GSALT units)
func (s *TransactionService) ProcessGiftIn(accountId string, amountGsaltUnits int64, description *string, externalRefId *string, giftSource *string) (*models.Transaction, error) {
	// Parse UUID first
	accountUUID, err := uuid.Parse(accountId)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid account ID format")
	}

	// Check idempotency
	if existingTxn, err := s.checkIdempotency(externalRefId); err != nil {
		return nil, err
	} else if existingTxn != nil {
		return existingTxn, nil // Return existing transaction
	}

	var transaction *models.Transaction
	now := time.Now()

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Lock and verify account with SELECT FOR UPDATE
		var account models.Account
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("connect_id = ?", accountUUID).First(&account).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.NewNotFoundError("Account not found")
			}
			return errors.NewInternalServerError(err, "Failed to get account")
		}

		// Create gift in transaction
		giftDescription := "Gift received"
		if description != nil {
			giftDescription = *description
		}
		if giftSource != nil {
			giftDescription += " from " + *giftSource
		}

		transaction = &models.Transaction{
			AccountID:           account.ConnectID,
			Type:                models.TransactionTypeGiftIn,
			AmountGsaltUnits:    amountGsaltUnits,
			Currency:            "GSALT",
			Status:              models.TransactionStatusCompleted,
			Description:         &giftDescription,
			ExternalReferenceID: externalRefId,
			CompletedAt:         &now,
		}

		if err := tx.Create(transaction).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create gift transaction")
		}

		// Update account balance atomically (both in GSALT units)
		account.Balance += amountGsaltUnits
		if err := tx.Save(&account).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to update account balance")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return transaction, nil
}

// ProcessGiftOut - Send gift to another account (in GSALT units)
func (s *TransactionService) ProcessGiftOut(sourceAccountId, destAccountId string, amountGsaltUnits int64, description *string) (*models.Transaction, *models.Transaction, error) {
	// Parse UUIDs first
	sourceUUID, err := uuid.Parse(sourceAccountId)
	if err != nil {
		return nil, nil, errors.NewBadRequestError("Invalid source account ID format")
	}

	destUUID, err := uuid.Parse(destAccountId)
	if err != nil {
		return nil, nil, errors.NewBadRequestError("Invalid destination account ID format")
	}

	// Check if trying to gift to same account
	if sourceUUID == destUUID {
		return nil, nil, errors.NewBadRequestError("Cannot send gift to the same account [" + ErrCodeSelfTransfer + "]")
	}

	var giftOut, giftIn *models.Transaction
	now := time.Now()

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Lock and verify source account with SELECT FOR UPDATE
		var sourceAccount models.Account
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("connect_id = ?", sourceUUID).First(&sourceAccount).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.NewNotFoundError("Source account not found")
			}
			return errors.NewInternalServerError(err, "Failed to get source account")
		}

		// Lock and verify destination account with SELECT FOR UPDATE
		var destAccount models.Account
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("connect_id = ?", destUUID).First(&destAccount).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.NewNotFoundError("Destination account not found")
			}
			return errors.NewInternalServerError(err, "Failed to get destination account")
		}

		// Check sufficient balance
		if !s.hasSufficientBalance(sourceAccount.Balance, amountGsaltUnits) {
			return errors.NewBadRequestError("Insufficient balance [" + ErrCodeInsufficientBalance + "]")
		}

		// Create gift out transaction
		giftOut = &models.Transaction{
			AccountID:            sourceAccount.ConnectID,
			Type:                 models.TransactionTypeGiftOut,
			AmountGsaltUnits:     amountGsaltUnits,
			Currency:             "GSALT",
			Status:               models.TransactionStatusCompleted,
			Description:          description,
			DestinationAccountID: &destAccount.ConnectID,
			CompletedAt:          &now,
		}

		if err := tx.Create(giftOut).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create gift out transaction")
		}

		// Create gift in transaction
		giftIn = &models.Transaction{
			AccountID:            destAccount.ConnectID,
			Type:                 models.TransactionTypeGiftIn,
			AmountGsaltUnits:     amountGsaltUnits,
			Currency:             "GSALT",
			Status:               models.TransactionStatusCompleted,
			Description:          description,
			SourceAccountID:      &sourceAccount.ConnectID,
			RelatedTransactionID: &giftOut.ID,
			CompletedAt:          &now,
		}

		if err := tx.Create(giftIn).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create gift in transaction")
		}

		// Update related transaction ID
		giftOut.RelatedTransactionID = &giftIn.ID
		if err := tx.Save(giftOut).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to update gift out transaction")
		}

		// Update account balances atomically (all in GSALT units)
		sourceAccount.Balance -= amountGsaltUnits
		destAccount.Balance += amountGsaltUnits

		if err := tx.Save(&sourceAccount).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to update source account balance")
		}

		if err := tx.Save(&destAccount).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to update destination account balance")
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return giftOut, giftIn, nil
}

// ConfirmPayment - Confirm payment for pending topup transactions
func (s *TransactionService) ConfirmPayment(transactionId string, externalPaymentId *string) (*models.Transaction, error) {
	transactionUUID, err := uuid.Parse(transactionId)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid transaction ID format")
	}

	var transaction *models.Transaction
	now := time.Now()

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Lock and get the transaction
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", transactionUUID).First(&transaction).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.NewNotFoundError("Transaction not found")
			}
			return errors.NewInternalServerError(err, "Failed to get transaction")
		}

		// Validate transaction can be confirmed
		if transaction.Status != models.TransactionStatusPending {
			return errors.NewBadRequestError("Only pending transactions can be confirmed [" + ErrCodeInvalidStatusTransition + "]")
		}

		if transaction.Type != models.TransactionTypeTopup {
			return errors.NewBadRequestError("Only topup transactions can be confirmed")
		}

		// Lock and get the account
		var account models.Account
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("connect_id = ?", transaction.AccountID).First(&account).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.NewNotFoundError("Account not found")
			}
			return errors.NewInternalServerError(err, "Failed to get account")
		}

		// Update transaction status
		transaction.Status = models.TransactionStatusCompleted
		transaction.CompletedAt = &now
		if externalPaymentId != nil {
			// You can store external payment ID in external_reference_id or create a new field
			if transaction.ExternalReferenceID == nil {
				transaction.ExternalReferenceID = externalPaymentId
			}
		}

		if err := tx.Save(transaction).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to update transaction")
		}

		// Update account balance
		account.Balance += transaction.AmountGsaltUnits
		if err := tx.Save(&account).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to update account balance")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return transaction, nil
}

// RejectPayment - Reject payment for pending transactions
func (s *TransactionService) RejectPayment(transactionId string, reason *string) (*models.Transaction, error) {
	transactionUUID, err := uuid.Parse(transactionId)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid transaction ID format")
	}

	var transaction models.Transaction

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Lock and get the transaction
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", transactionUUID).First(&transaction).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.NewNotFoundError("Transaction not found")
			}
			return errors.NewInternalServerError(err, "Failed to get transaction")
		}

		// Validate transaction can be rejected
		if transaction.Status != models.TransactionStatusPending {
			return errors.NewBadRequestError("Only pending transactions can be rejected [" + ErrCodeInvalidStatusTransition + "]")
		}

		// Update transaction status
		transaction.Status = models.TransactionStatusFailed
		if reason != nil && *reason != "" {
			failureReason := fmt.Sprintf("Payment rejected: %s", *reason)
			transaction.Description = &failureReason
		}

		if err := tx.Save(&transaction).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to update transaction")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &transaction, nil
}
