package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/app/pkg"
	"github.com/safatanc/gsalt-core/internal/infrastructures"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

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
	ErrCodeInvalidPaymentMethod    = "INVALID_PAYMENT_METHOD"
)

// Supported currencies for payments
var supportedCurrencies = map[string]bool{
	"GSALT": true,
	"IDR":   true,
	"USD":   true,
	"EUR":   true,
	"SGD":   true,
}

type TransactionService struct {
	db                   *gorm.DB
	validator            *infrastructures.Validator
	accountService       *AccountService
	flipService          *FlipService
	connectService       *ConnectService
	paymentMethodService *PaymentMethodService
	auditService         *AuditService
	limits               TransactionLimits
}

func NewTransactionService(
	db *gorm.DB,
	validator *infrastructures.Validator,
	accountService *AccountService,
	flipService *FlipService,
	connectService *ConnectService,
	paymentMethodService *PaymentMethodService,
	auditService *AuditService,
) *TransactionService {
	return &TransactionService{
		db:                   db,
		validator:            validator,
		accountService:       accountService,
		flipService:          flipService,
		connectService:       connectService,
		paymentMethodService: paymentMethodService,
		auditService:         auditService,
		limits:               defaultLimits,
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

// Helper function to parse UUID with better error handling
func (s *TransactionService) parseUUID(id, fieldName string) (uuid.UUID, error) {
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, errors.NewBadRequestError(fmt.Sprintf("Invalid %s format", fieldName))
	}
	return parsedUUID, nil
}

// Helper function to parse optional UUID
func (s *TransactionService) parseOptionalUUID(id *string, fieldName string) (*uuid.UUID, error) {
	if id == nil {
		return nil, nil
	}
	parsedUUID, err := uuid.Parse(*id)
	if err != nil {
		return nil, errors.NewBadRequestError(fmt.Sprintf("Invalid %s format", fieldName))
	}
	return &parsedUUID, nil
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

func (s *TransactionService) CreateTransaction(req *models.TransactionCreateRequest) (*models.Transaction, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	// Parse UUID fields using helper function
	accountUUID, err := s.parseUUID(req.AccountID, "account ID")
	if err != nil {
		return nil, err
	}

	relatedTransactionUUID, err := s.parseOptionalUUID(req.RelatedTransactionID, "related transaction ID")
	if err != nil {
		return nil, err
	}

	sourceAccountUUID, err := s.parseOptionalUUID(req.SourceAccountID, "source account ID")
	if err != nil {
		return nil, err
	}

	destinationAccountUUID, err := s.parseOptionalUUID(req.DestinationAccountID, "destination account ID")
	if err != nil {
		return nil, err
	}

	// Create transaction - GORM will handle CreatedAt and UpdatedAt automatically
	transaction := &models.Transaction{
		AccountID:             accountUUID,
		Type:                  req.Type,
		AmountGsaltUnits:      req.AmountGsaltUnits,
		FeeGsaltUnits:         0,
		TotalAmountGsaltUnits: req.AmountGsaltUnits,
		Currency:              req.Currency,
		ExchangeRateIDR:       req.ExchangeRateIDR,
		PaymentAmount:         req.PaymentAmount,
		PaymentCurrency:       req.PaymentCurrency,
		PaymentMethod:         req.PaymentMethod,
		Status:                models.TransactionStatusPending,
		Description:           req.Description,
		RelatedTransactionID:  relatedTransactionUUID,
		SourceAccountID:       sourceAccountUUID,
		DestinationAccountID:  destinationAccountUUID,
		VoucherCode:           req.VoucherCode,
		ExternalReferenceID:   req.ExternalReferenceID,
		PaymentInstructions:   req.PaymentInstructions,
	}

	if err := s.db.Create(transaction).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to create transaction")
	}

	return transaction, nil
}

func (s *TransactionService) GetTransaction(transactionId string) (*models.Transaction, error) {
	transactionUUID, err := s.parseUUID(transactionId, "transaction ID")
	if err != nil {
		return nil, err
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

func (s *TransactionService) GetTransactionByRef(ref string) (*models.TransactionResponse, error) {
	var transaction models.Transaction
	err := s.db.Where("external_reference_id = ?", ref).First(&transaction).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Transaction not found")
		}
		return nil, errors.NewInternalServerError(err, "Failed to get transaction")
	}

	paymentInstructions, _ := transaction.GetPaymentInstructions()

	return &models.TransactionResponse{
		Transaction:         &transaction,
		PaymentInstructions: paymentInstructions,
	}, nil
}

func (s *TransactionService) GetTransactionsByAccount(accountId string, pagination *models.PaginationRequest) (*models.Pagination[[]models.Transaction], error) {
	accountUUID, err := s.parseUUID(accountId, "account ID")
	if err != nil {
		return nil, err
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

func (s *TransactionService) UpdateTransaction(transactionId string, req *models.TransactionUpdateRequest) (*models.Transaction, error) {
	// Validate request
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	// Parse transaction ID
	transactionUUID, err := s.parseUUID(transactionId, "transaction ID")
	if err != nil {
		return nil, err
	}

	// Find existing transaction
	var transaction models.Transaction
	if err := s.db.First(&transaction, "id = ?", transactionUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Transaction not found")
		}
		return nil, errors.NewInternalServerError(err, "Failed to find transaction")
	}

	// Store old state for audit
	oldTransaction := transaction

	// Parse UUID fields if provided
	relatedTransactionUUID, err := s.parseOptionalUUID(req.RelatedTransactionID, "related transaction ID")
	if err != nil {
		return nil, err
	}

	sourceAccountUUID, err := s.parseOptionalUUID(req.SourceAccountID, "source account ID")
	if err != nil {
		return nil, err
	}

	destinationAccountUUID, err := s.parseOptionalUUID(req.DestinationAccountID, "destination account ID")
	if err != nil {
		return nil, err
	}

	// Update fields - GORM will handle UpdatedAt automatically
	updates := map[string]interface{}{}

	if req.Status != nil {
		oldStatus := transaction.Status
		updates["status"] = *req.Status
		if *req.Status == models.TransactionStatusCompleted {
			now := time.Now()
			updates["completed_at"] = &now
		}

		// Log status change
		metadata := map[string]interface{}{
			"payment_method":           transaction.PaymentMethod,
			"external_payment_id":      transaction.ExternalPaymentID,
			"processing_status":        transaction.Status,
			"amount_gsalt_units":       transaction.AmountGsaltUnits,
			"fee_gsalt_units":          transaction.FeeGsaltUnits,
			"total_amount_gsalt_units": transaction.TotalAmountGsaltUnits,
		}

		if err := s.auditService.LogTransactionStatusChange(
			transaction.ID,
			oldStatus,
			*req.Status,
			"Status updated via UpdateTransaction",
			metadata,
			nil, // TODO: Add user ID from context
		); err != nil {
			return nil, err
		}
	}
	if req.ExchangeRateIDR != nil {
		updates["exchange_rate_idr"] = req.ExchangeRateIDR
	}
	if req.PaymentAmount != nil {
		updates["payment_amount"] = req.PaymentAmount
	}
	if req.PaymentCurrency != nil {
		updates["payment_currency"] = req.PaymentCurrency
	}
	if req.PaymentMethod != nil {
		updates["payment_method"] = req.PaymentMethod
	}
	if req.PaymentInstructions != nil {
		updates["payment_instructions"] = req.PaymentInstructions
	}
	if req.Description != nil {
		updates["description"] = req.Description
	}
	if req.RelatedTransactionID != nil {
		updates["related_transaction_id"] = relatedTransactionUUID
	}
	if req.SourceAccountID != nil {
		updates["source_account_id"] = sourceAccountUUID
	}
	if req.DestinationAccountID != nil {
		updates["destination_account_id"] = destinationAccountUUID
	}
	if req.VoucherCode != nil {
		updates["voucher_code"] = req.VoucherCode
	}
	if req.ExternalReferenceID != nil {
		updates["external_reference_id"] = req.ExternalReferenceID
	}
	if req.CompletedAt != nil {
		updates["completed_at"] = req.CompletedAt
	}

	// Apply updates
	if err := s.db.Model(&transaction).Updates(updates).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to update transaction")
	}

	// Log the entire change
	if err := s.auditService.LogAudit(
		"transactions",
		transaction.ID,
		models.AuditActionUpdate,
		oldTransaction,
		transaction,
		nil, // TODO: Add user ID from context
	); err != nil {
		return nil, err
	}

	return &transaction, nil
}

// Helper function to create transaction with common fields
func (s *TransactionService) createBaseTransaction(accountUUID uuid.UUID, txnType models.TransactionType, amountGsaltUnits int64, status models.TransactionStatus, description *string) *models.Transaction {
	externalRefId := "GSALT-" + pkg.RandomNumberString(8)
	return &models.Transaction{
		AccountID:             accountUUID,
		ExternalReferenceID:   &externalRefId,
		Type:                  txnType,
		AmountGsaltUnits:      amountGsaltUnits,
		FeeGsaltUnits:         0,
		TotalAmountGsaltUnits: amountGsaltUnits,
		Currency:              "GSALT",
		Status:                status,
		Description:           description,
	}
}

func (s *TransactionService) ProcessTopup(accountId string, amountGsaltUnits int64, paymentAmount *int64, paymentCurrency *string, paymentMethodCode *string, externalRefId *string) (*models.TransactionResponse, error) {
	// Parse UUID first
	accountUUID, err := s.parseUUID(accountId, "account ID")
	if err != nil {
		return nil, err
	}

	// Validate payment method code is provided
	if paymentMethodCode == nil || *paymentMethodCode == "" {
		return nil, errors.NewBadRequestError("Payment method code is required")
	}

	// Get Payment Method configuration from the database
	paymentMethod, err := s.paymentMethodService.FindByCode(*paymentMethodCode)
	if err != nil {
		return nil, err // The error from the service is already well-formatted
	}

	// Check if the method is available for top-up
	if !paymentMethod.IsAvailableForTopup {
		return nil, errors.NewBadRequestError(fmt.Sprintf("Payment method '%s' is not available for top-up", *paymentMethodCode))
	}

	// Validate transaction amount
	if err := s.validateTransactionAmount(models.TransactionTypeTopup, amountGsaltUnits); err != nil {
		return nil, err
	}

	// Check idempotency
	if existingTxn, err := s.checkIdempotency(externalRefId); err != nil {
		return nil, err
	} else if existingTxn != nil {
		paymentInstructions, _ := existingTxn.GetPaymentInstructions()
		return &models.TransactionResponse{
			Transaction:         existingTxn,
			PaymentInstructions: paymentInstructions,
		}, nil
	}

	// Check daily limits
	if err := s.checkDailyLimits(accountUUID, models.TransactionTypeTopup, amountGsaltUnits); err != nil {
		return nil, err
	}

	// Auto-calculate payment_amount if not provided
	var finalPaymentAmount int64
	if paymentAmount != nil {
		finalPaymentAmount = *paymentAmount
	} else {
		finalPaymentAmount = s.ConvertGSALTToIDR(amountGsaltUnits)
	}

	var transaction *models.Transaction
	exchangeRate := decimal.NewFromInt(1000) // 1 GSALT = 1000 IDR
	ctx := context.Background()

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Create transaction record
		description := fmt.Sprintf("Topup %d GSALT", amountGsaltUnits/100)
		transaction = s.createBaseTransaction(accountUUID, models.TransactionTypeTopup, amountGsaltUnits, models.TransactionStatusPending, &description)
		transaction.ExchangeRateIDR = &exchangeRate
		transaction.PaymentAmount = &finalPaymentAmount
		transaction.PaymentCurrency = &paymentMethod.Currency
		transaction.PaymentMethod = &paymentMethod.Code
		if externalRefId != nil {
			transaction.ExternalReferenceID = externalRefId
		}

		if err := tx.Create(transaction).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create topup transaction")
		}

		// --- PAYLOAD TO PROVIDER ---
		// Currently, we only support FLIP. In the future, we can use a factory or strategy pattern here based on `paymentMethod.ProviderCode`.
		if paymentMethod.ProviderCode != "FLIP" {
			return errors.NewInternalServerError(nil, fmt.Sprintf("Payment provider '%s' is not implemented", paymentMethod.ProviderCode))
		}

		connectUser, err := s.connectService.GetUser(accountUUID.String())
		if err != nil {
			return errors.NewInternalServerError(err, "Failed to get connect user details")
		}

		internalRef := fmt.Sprintf("GSALT-%s", pkg.RandomNumberString(10))

		billReq := models.CreateBillRequest{
			Title:                 fmt.Sprintf("GSALT Topup - %s", internalRef),
			Type:                  "SINGLE",
			Amount:                finalPaymentAmount,
			RedirectURL:           s.flipService.client.GetDefaultRedirectURL(),
			IsPhoneNumberRequired: true,
			Step:                  "3",
			SenderName:            connectUser.FullName,
			SenderEmail:           connectUser.Email,
			SenderPhoneNumber:     "081234567890", // Placeholder
			SenderBank:            paymentMethod.ProviderMethodCode,
			SenderBankType:        paymentMethod.ProviderMethodType,
			ReferenceID:           internalRef,
			ChargeFee:             false, // We handle fee display, Flip doesn't need to charge the user
		}

		billResp, err := s.flipService.CreateBill(ctx, billReq)
		if err != nil {
			return fmt.Errorf("failed to create bill with provider: %w", err)
		}

		instructionsJSON, err := json.Marshal(billResp)
		if err != nil {
			return fmt.Errorf("failed to marshal payment instructions: %w", err)
		}
		instructionsJSONStr := string(instructionsJSON)

		transaction.ExternalReferenceID = &internalRef
		linkIDStr := fmt.Sprintf("%d", billResp.LinkID)
		transaction.ExternalPaymentID = &linkIDStr
		transaction.PaymentInstructions = &instructionsJSONStr

		if err := tx.Save(transaction).Error; err != nil {
			return fmt.Errorf("failed to update transaction with payment details: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	paymentInstructions, _ := transaction.GetPaymentInstructions()
	response := &models.TransactionResponse{
		Transaction:         transaction,
		PaymentInstructions: paymentInstructions,
	}

	return response, nil
}

// updateAccountBalance updates the account balance with the given amount
func (s *TransactionService) updateAccountBalance(tx *gorm.DB, accountID uuid.UUID, amountGsaltUnits int64) error {
	// Lock and get account
	var account models.Account
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("connect_id = ?", accountID).First(&account).Error; err != nil {
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
	// Parse UUIDs using helper functions
	sourceUUID, err := s.parseUUID(sourceAccountId, "source account ID")
	if err != nil {
		return nil, nil, err
	}

	destUUID, err := s.parseUUID(destAccountId, "destination account ID")
	if err != nil {
		return nil, nil, err
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

		// Create outgoing transfer record using helper function
		transferOut = s.createBaseTransaction(sourceUUID, models.TransactionTypeTransferOut, amountGsaltUnits, models.TransactionStatusCompleted, description)
		transferOut.DestinationAccountID = &destUUID
		transferOut.CompletedAt = &now

		// Create transaction in database
		if err := tx.Create(transferOut).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create transfer out transaction")
		}

		// Create incoming transfer record with reference to outgoing transfer
		transferOutIDPtr := transferOut.ID
		transferIn = s.createBaseTransaction(destUUID, models.TransactionTypeTransferIn, amountGsaltUnits, models.TransactionStatusCompleted, description)
		transferIn.SourceAccountID = &sourceUUID
		transferIn.RelatedTransactionID = &transferOutIDPtr
		transferIn.CompletedAt = &now

		// Create transaction in database
		if err := tx.Create(transferIn).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create transfer in transaction")
		}

		// Update related transaction ID in transfer out record
		transferInIDPtr := transferIn.ID
		transferOut.RelatedTransactionID = &transferInIDPtr

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

// ProcessPayment processes external payment (QRIS, Bank Transfer, E-wallet, Credit Card, etc.)
func (s *TransactionService) ProcessPayment(request models.PaymentRequest) (models.PaymentResponse, error) {
	// Validate request
	if err := s.validator.Validate(request); err != nil {
		return models.PaymentResponse{}, err
	}

	ctx := context.Background()

	// Get ConnectUser directly
	connectUser, err := s.connectService.GetUser(request.AccountID)
	if err != nil {
		return models.PaymentResponse{}, fmt.Errorf("connect user not found: %w", err)
	}

	// Get Payment Method configuration from DB
	paymentMethod, err := s.paymentMethodService.FindByCode(request.PaymentMethod)
	if err != nil {
		return models.PaymentResponse{}, err
	}
	if !paymentMethod.IsAvailableForTopup { // Payments are a form of topup/funding
		return models.PaymentResponse{}, errors.NewBadRequestError(fmt.Sprintf("Payment method '%s' is not available for payment", request.PaymentMethod))
	}

	amountInIDR := s.ConvertGSALTToIDR(request.AmountGsaltUnits)
	feeDecimal := s.paymentMethodService.CalculateFee(*paymentMethod, decimal.NewFromInt(amountInIDR))
	feeGsaltUnits := s.ConvertIDRToGSALT(feeDecimal.IntPart())

	// Total amount including fee
	totalAmountGsalt := request.AmountGsaltUnits + feeGsaltUnits
	totalAmountIDR := s.ConvertGSALTToIDR(totalAmountGsalt)

	// Parse account UUID
	accountUUID, err := s.parseUUID(request.AccountID, "account ID")
	if err != nil {
		return models.PaymentResponse{}, fmt.Errorf("invalid account ID format: %w", err)
	}

	// Create payment transaction
	description := fmt.Sprintf("Payment via %s", request.PaymentMethod)
	transaction := s.createBaseTransaction(accountUUID, models.TransactionTypePayment, request.AmountGsaltUnits, models.TransactionStatusPending, &description)
	transaction.FeeGsaltUnits = feeGsaltUnits
	transaction.TotalAmountGsaltUnits = totalAmountGsalt
	transaction.PaymentMethod = &request.PaymentMethod
	transaction.PaymentAmount = &totalAmountIDR
	transaction.PaymentCurrency = &paymentMethod.Currency

	if err := s.db.Create(&transaction).Error; err != nil {
		return models.PaymentResponse{}, fmt.Errorf("failed to create transaction: %w", err)
	}

	// --- Create Bill via Provider ---
	if paymentMethod.ProviderCode != "FLIP" {
		return models.PaymentResponse{}, errors.NewInternalServerError(nil, "Provider not implemented")
	}

	internalRef := *transaction.ExternalReferenceID
	billReq := models.CreateBillRequest{
		Title:                 fmt.Sprintf("GSALT Payment - %s", internalRef),
		Type:                  "SINGLE",
		Amount:                totalAmountIDR,
		RedirectURL:           s.flipService.client.GetDefaultRedirectURL(),
		IsPhoneNumberRequired: true,
		Step:                  "3",
		SenderName:            connectUser.FullName,
		SenderEmail:           connectUser.Email,
		SenderPhoneNumber:     "081234567890", // Placeholder
		ReferenceID:           internalRef,
		SenderBank:            paymentMethod.ProviderMethodCode,
		SenderBankType:        paymentMethod.ProviderMethodType,
	}

	billResp, err := s.flipService.CreateBill(ctx, billReq)
	if err != nil {
		s.db.Delete(&transaction) // Rollback
		return models.PaymentResponse{}, fmt.Errorf("failed to create Flip bill: %w", err)
	}

	instructionsJSON, err := json.Marshal(billResp)
	if err != nil {
		s.db.Delete(&transaction) // Rollback
		return models.PaymentResponse{}, fmt.Errorf("failed to marshal payment instructions: %w", err)
	}
	instructionsJSONStr := string(instructionsJSON)
	linkIDStr := fmt.Sprintf("%d", billResp.LinkID)

	transaction.ExternalPaymentID = &linkIDStr
	transaction.PaymentInstructions = &instructionsJSONStr
	if err := s.db.Save(&transaction).Error; err != nil {
		return models.PaymentResponse{}, fmt.Errorf("failed to update transaction: %w", err)
	}

	// --- Construct Final Response ---
	paymentURL := billResp.PaymentURL
	if paymentURL == "" {
		paymentURL = billResp.LinkURL
	}

	return models.PaymentResponse{
		TransactionID:       transaction.ID.String(),
		Status:              string(transaction.Status),
		PaymentMethod:       request.PaymentMethod,
		Amount:              request.AmountGsaltUnits,
		Fee:                 feeGsaltUnits,
		TotalAmount:         totalAmountGsalt,
		PaymentURL:          paymentURL,
		ExpiryTime:          pkg.ParseFlipExpiryTime(billResp.ExpiredDate),
		PaymentInstructions: billResp.Instructions,
	}, nil
}

// Helper function to validate transaction amounts (in GSALT units)
func (s *TransactionService) validateTransactionAmount(txnType models.TransactionType, amountGsaltUnits int64) error {
	switch txnType {
	case models.TransactionTypeTopup:
		if amountGsaltUnits < s.limits.MinTopupAmount {
			return fmt.Errorf("topup amount must be at least %d GSALT units", s.limits.MinTopupAmount)
		}
		if amountGsaltUnits > s.limits.MaxTopupAmount {
			return fmt.Errorf("topup amount cannot exceed %d GSALT units", s.limits.MaxTopupAmount)
		}
	case models.TransactionTypeTransferOut:
		if amountGsaltUnits < s.limits.MinTransferAmount {
			return fmt.Errorf("transfer amount must be at least %d GSALT units", s.limits.MinTransferAmount)
		}
		if amountGsaltUnits > s.limits.MaxTransferAmount {
			return fmt.Errorf("transfer amount cannot exceed %d GSALT units", s.limits.MaxTransferAmount)
		}
	case models.TransactionTypePayment:
		if amountGsaltUnits < s.limits.MinPaymentAmount {
			return fmt.Errorf("payment amount must be at least %d GSALT units", s.limits.MinPaymentAmount)
		}
		if amountGsaltUnits > s.limits.MaxPaymentAmount {
			return fmt.Errorf("payment amount cannot exceed %d GSALT units", s.limits.MaxPaymentAmount)
		}
	case models.TransactionTypeWithdrawal:
		if amountGsaltUnits < s.limits.MinTransferAmount {
			return fmt.Errorf("withdrawal amount must be at least %d GSALT units", s.limits.MinTransferAmount)
		}
		if amountGsaltUnits > s.limits.MaxTransferAmount {
			return fmt.Errorf("withdrawal amount cannot exceed %d GSALT units", s.limits.MaxTransferAmount)
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
	case models.TransactionTypeWithdrawal:
		dailyLimit = defaultLimits.DailyTransferLimit
		s.db.Model(&models.Transaction{}).
			Where("account_id = ? AND type = ? AND created_at >= ? AND created_at < ? AND status = ?",
				accountId, models.TransactionTypeWithdrawal, today, tomorrow, models.TransactionStatusCompleted).
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
			"status": models.TransactionStatusCancelled,
			// GORM will handle updated_at automatically
		})

	if result.Error != nil {
		return errors.NewInternalServerError(result.Error, "Failed to expire pending transactions")
	}

	return nil
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
	// Parse UUID using helper function
	accountUUID, err := s.parseUUID(accountId, "account ID")
	if err != nil {
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

		transaction = s.createBaseTransaction(accountUUID, models.TransactionTypeGiftIn, amountGsaltUnits, models.TransactionStatusCompleted, &giftDescription)
		transaction.ExternalReferenceID = externalRefId
		transaction.CompletedAt = &now

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
	// Parse UUIDs using helper functions
	sourceUUID, err := s.parseUUID(sourceAccountId, "source account ID")
	if err != nil {
		return nil, nil, err
	}

	destUUID, err := s.parseUUID(destAccountId, "destination account ID")
	if err != nil {
		return nil, nil, err
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

		// Create gift out transaction using helper function
		giftOut = s.createBaseTransaction(sourceUUID, models.TransactionTypeGiftOut, amountGsaltUnits, models.TransactionStatusCompleted, description)
		giftOut.DestinationAccountID = &destUUID
		giftOut.CompletedAt = &now

		if err := tx.Create(giftOut).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create gift out transaction")
		}

		// Create gift in transaction using helper function
		giftIn = s.createBaseTransaction(destUUID, models.TransactionTypeGiftIn, amountGsaltUnits, models.TransactionStatusCompleted, description)
		giftIn.SourceAccountID = &sourceUUID
		giftIn.RelatedTransactionID = &giftOut.ID
		giftIn.CompletedAt = &now

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

// ProcessWithdrawal processes GSALT withdrawal to bank account via Flip disbursement
func (s *TransactionService) ProcessWithdrawal(accountId string, amountGsaltUnits int64, bankCode, accountNumber, recipientName string, description *string, externalRefId *string) (*models.Transaction, error) {
	// Parse account UUID using helper function
	accountUUID, err := s.parseUUID(accountId, "account ID")
	if err != nil {
		return nil, fmt.Errorf("invalid account ID format: %w", err)
	}

	// Get account to validate balance
	var account models.Account
	if err := s.db.Where("connect_id = ?", accountUUID).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Account not found")
		}
		return nil, errors.NewInternalServerError(err, "Failed to get account")
	}

	// Check if account has sufficient balance
	if !s.hasSufficientBalance(account.Balance, amountGsaltUnits) {
		return nil, errors.NewBadRequestError("Insufficient balance [" + ErrCodeInsufficientBalance + "]")
	}

	// Validate transaction amount
	if err := s.validateTransactionAmount(models.TransactionTypeWithdrawal, amountGsaltUnits); err != nil {
		return nil, errors.NewBadRequestError(err.Error())
	}

	// Check daily limits
	if err := s.checkDailyLimits(accountUUID, models.TransactionTypeWithdrawal, amountGsaltUnits); err != nil {
		return nil, err
	}

	// Validate bank account first
	ctx := context.Background()
	bankInquiry, err := s.flipService.BankAccountInquiry(ctx, models.BankAccountInquiryRequest{
		AccountNumber: accountNumber,
		BankCode:      bankCode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to validate bank account: %w", err)
	}

	if bankInquiry == nil {
		return nil, fmt.Errorf("invalid bank account")
	}

	// Check disbursement service availability
	available, err := s.CheckDisbursementAvailability(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check disbursement availability: %w", err)
	}

	if !available {
		return nil, errors.NewBadRequestError("Disbursement service is currently unavailable")
	}

	var transaction *models.Transaction

	// Execute withdrawal in database transaction
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Create withdrawal transaction using helper function
		withdrawalDesc := fmt.Sprintf("Withdrawal %d GSALT to %s (%s)", amountGsaltUnits/100, bankCode, accountNumber)
		if description != nil {
			withdrawalDesc = *description
		}

		transaction = s.createBaseTransaction(accountUUID, models.TransactionTypeWithdrawal, amountGsaltUnits, models.TransactionStatusPending, &withdrawalDesc)
		if externalRefId != nil {
			transaction.ExternalReferenceID = externalRefId
		}

		if err := tx.Create(transaction).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create withdrawal transaction")
		}

		// Deduct balance immediately (will be reversed if disbursement fails)
		if err := s.updateAccountBalance(tx, accountUUID, -amountGsaltUnits); err != nil {
			return err
		}

		// Convert GSALT to IDR for disbursement
		amountIDR := s.ConvertGSALTToIDR(amountGsaltUnits)

		// Create disbursement request
		transactionIDStr := transaction.ID.String()
		disbursementReq := models.DisbursementRequest{
			AccountNumber:  accountNumber,
			BankCode:       bankCode,
			Amount:         amountIDR,
			Remark:         fmt.Sprintf("GSALT Withdrawal - %s", transactionIDStr),
			IdempotencyKey: transactionIDStr,
			Timestamp:      time.Now().Format(time.RFC3339),
		}

		// Create disbursement via Flip
		disbursementResp, err := s.flipService.CreateDisbursement(ctx, disbursementReq)
		if err != nil {
			// If disbursement fails, we need to reverse the balance deduction
			reverseTx := s.db.Begin()
			s.updateAccountBalance(reverseTx, accountUUID, amountGsaltUnits)
			reverseTx.Commit()

			return fmt.Errorf("failed to create disbursement: %w", err)
		}

		// Update transaction with disbursement ID
		disbursementID := fmt.Sprintf("%d", disbursementResp.ID)
		transaction.ExternalPaymentID = &disbursementID

		if err := tx.Save(transaction).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to update transaction with disbursement ID")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return transaction, nil
}

// GetSupportedBanksForWithdrawal retrieves supported banks for withdrawal
func (s *TransactionService) GetSupportedBanksForWithdrawal(ctx context.Context) ([]models.BankListResponse, error) {
	flipBanks, err := s.GetSupportedBanksForFlip(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get supported banks: %w", err)
	}

	// Convert to our response format
	var banks []models.BankListResponse
	for _, bank := range flipBanks {
		banks = append(banks, models.BankListResponse{
			BankCode: bank.BankCode,
			BankName: bank.Name,
		})
	}

	return banks, nil
}

// ValidateBankAccountForWithdrawal validates a bank account for withdrawal
func (s *TransactionService) ValidateBankAccountForWithdrawal(ctx context.Context, bankCode, accountNumber string) (*models.BankAccountInquiryResponse, error) {
	return s.ValidateBankAccountForFlip(ctx, accountNumber, bankCode)
}

// GetWithdrawalBalance gets available balance for withdrawal (considering pending withdrawals)
func (s *TransactionService) GetWithdrawalBalance(accountId string) (int64, error) {
	accountUUID, err := s.parseUUID(accountId, "account ID")
	if err != nil {
		return 0, fmt.Errorf("invalid account ID: %w", err)
	}

	// Get account balance
	var account models.Account
	if err := s.db.First(&account, "connect_id = ?", accountUUID).Error; err != nil {
		return 0, fmt.Errorf("account not found: %w", err)
	}

	// Calculate pending withdrawal amount
	var pendingWithdrawals int64
	s.db.Model(&models.Transaction{}).
		Where("account_id = ? AND type = ? AND status IN (?)",
			accountUUID, models.TransactionTypeWithdrawal,
			[]models.TransactionStatus{models.TransactionStatusPending, models.TransactionStatusProcessing}).
		Select("COALESCE(SUM(amount_gsalt_units), 0)").Scan(&pendingWithdrawals)

	// Available balance = total balance - pending withdrawals
	availableBalance := account.Balance - pendingWithdrawals
	if availableBalance < 0 {
		availableBalance = 0
	}

	return availableBalance, nil
}

// getPaymentInstructions returns payment instructions based on payment method
func (s *TransactionService) getPaymentInstructions(paymentMethod string) string {
	switch {
	case strings.HasPrefix(paymentMethod, "VA_"):
		return "Silakan transfer ke nomor Virtual Account yang telah disediakan melalui ATM, mobile banking, atau internet banking."
	case paymentMethod == "QRIS":
		return "Scan kode QR menggunakan aplikasi pembayaran digital Anda (GoPay, OVO, DANA, ShopeePay, dll)."
	case strings.HasPrefix(paymentMethod, "EWALLET_"):
		return "Bayar menggunakan aplikasi e-wallet yang dipilih. Anda akan diarahkan ke aplikasi tersebut."
	case paymentMethod == "CREDIT_CARD" || paymentMethod == "DEBIT_CARD":
		return "Masukkan informasi kartu kredit/debit Anda untuk menyelesaikan pembayaran."
	case strings.HasPrefix(paymentMethod, "RETAIL_"):
		return "Kunjungi outlet retail yang dipilih dan berikan kode pembayaran kepada kasir."
	case paymentMethod == "DIRECT_DEBIT":
		return "Pembayaran akan langsung didebit dari rekening bank yang terdaftar."
	default:
		return "Ikuti instruksi pembayaran yang tersedia di halaman checkout."
	}
}

// ValidateBankAccountForFlip validates bank account for withdrawal using Flip API
func (s *TransactionService) ValidateBankAccountForFlip(ctx context.Context, accountNumber, bankCode string) (*models.BankAccountInquiryResponse, error) {
	req := models.BankAccountInquiryRequest{
		AccountNumber: accountNumber,
		BankCode:      bankCode,
	}
	return s.flipService.BankAccountInquiry(ctx, req)
}

// CheckDisbursementAvailability checks if disbursement service is available
func (s *TransactionService) CheckDisbursementAvailability(ctx context.Context) (bool, error) {
	// Check maintenance info to see if disbursement is available
	maintenance, err := s.flipService.GetMaintenanceInfo(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get maintenance info: %w", err)
	}

	// Check if disbursement service is under maintenance
	// This is a simplified check - you might need to adjust based on actual maintenance response structure
	return maintenance != nil, nil
}

// ConvertGSALTToIDR converts GSALT units to IDR
func (s *TransactionService) ConvertGSALTToIDR(gsaltUnits int64) int64 {
	// 1 GSALT = 1000 IDR (based on your existing logic)
	// gsaltUnits are in the smallest unit (1 GSALT = 100 units)
	// So, (gsaltUnits / 100) * 1000 = gsaltUnits * 10
	return gsaltUnits * 10
}

// ConvertIDRToGSALT converts IDR to GSALT
func (s *TransactionService) ConvertIDRToGSALT(idrAmount int64) int64 {
	// 1 GSALT = 1000 IDR, 1 GSALT = 100 units
	// idrAmount / 1000 * 100 = idrAmount / 10
	return idrAmount / 10
}

// CreateGSALTDisbursement creates a disbursement for GSALT withdrawal
func (s *TransactionService) CreateGSALTDisbursement(ctx context.Context, req models.DisbursementRequest) (*models.DisbursementResponse, error) {
	return s.flipService.CreateDisbursement(ctx, req)
}

// GetSupportedBanksForFlip returns list of supported banks for disbursement from Flip
func (s *TransactionService) GetSupportedBanksForFlip(ctx context.Context) ([]models.FlipBank, error) {
	return s.flipService.GetBanks(ctx)
}

// CheckWithdrawalStatus checks the status of a withdrawal transaction
func (s *TransactionService) CheckWithdrawalStatus(transactionId string) (*models.WithdrawalResponse, error) {
	// Get transaction
	transaction, err := s.GetTransaction(transactionId)
	if err != nil {
		return nil, err
	}

	if transaction.Type != models.TransactionTypeWithdrawal {
		return nil, errors.NewBadRequestError("Transaction is not a withdrawal")
	}

	// Get disbursement status from Flip if we have external payment ID
	var disbursementStatus *models.DisbursementResponse
	if transaction.ExternalPaymentID != nil {
		ctx := context.Background()
		disbursementStatus, err = s.flipService.GetDisbursementByID(ctx, *transaction.ExternalPaymentID)
		if err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to get disbursement status: %v\n", err)
		}
	}

	response := &models.WithdrawalResponse{
		Transaction:   transaction,
		Status:        string(transaction.Status),
		EstimatedTime: "1-3 business days",
	}

	if disbursementStatus != nil {
		disbursementIDStr := fmt.Sprintf("%d", disbursementStatus.ID)
		response.DisbursementID = &disbursementIDStr
	}

	return response, nil
}

// ConfirmPayment - Confirm payment for pending topup transactions
func (s *TransactionService) ConfirmPayment(transactionId string, externalPaymentId *string) (*models.Transaction, error) {
	transactionUUID, err := s.parseUUID(transactionId, "transaction ID")
	if err != nil {
		return nil, err
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
	transactionUUID, err := s.parseUUID(transactionId, "transaction ID")
	if err != nil {
		return nil, err
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

// DeleteTransaction performs soft delete on a transaction
func (s *TransactionService) DeleteTransaction(transactionId string) error {
	transactionUUID, err := s.parseUUID(transactionId, "transaction ID")
	if err != nil {
		return err
	}

	var transaction models.Transaction
	if err := s.db.First(&transaction, "id = ?", transactionUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewNotFoundError("Transaction not found")
		}
		return errors.NewInternalServerError(err, "Failed to find transaction")
	}

	// Store transaction state before deletion
	deletedTransaction := transaction

	// Perform soft delete
	if err := s.db.Delete(&transaction).Error; err != nil {
		return errors.NewInternalServerError(err, "Failed to delete transaction")
	}

	// Log the deletion
	if err := s.auditService.LogAudit(
		"transactions",
		transaction.ID,
		models.AuditActionDelete,
		deletedTransaction,
		nil,
		nil, // TODO: Add user ID from context
	); err != nil {
		return err
	}

	return nil
}
