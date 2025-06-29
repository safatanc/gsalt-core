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

// Flip payment method constants for compatibility
const (
	FlipPaymentMethodQRIS           = "QRIS"
	FlipPaymentMethodVirtualAccount = "VA"
	FlipPaymentMethodEWallet        = "EWALLET"
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
	db             *gorm.DB
	validator      *infrastructures.Validator
	accountService *AccountService
	flipService    *FlipService
	connectService *ConnectService
	limits         TransactionLimits
}

func NewTransactionService(db *gorm.DB, validator *infrastructures.Validator, accountService *AccountService, flipService *FlipService, connectService *ConnectService) *TransactionService {
	return &TransactionService{
		db:             db,
		validator:      validator,
		accountService: accountService,
		flipService:    flipService,
		connectService: connectService,
		limits:         defaultLimits,
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
		updates["status"] = *req.Status
		if *req.Status == models.TransactionStatusCompleted {
			now := time.Now()
			updates["completed_at"] = &now
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

func (s *TransactionService) ProcessTopup(accountId string, amountGsaltUnits int64, paymentAmount *int64, paymentCurrency *string, paymentMethod *string, externalRefId *string) (*models.TopupResponse, error) {
	// Parse UUID first
	accountUUID, err := s.parseUUID(accountId, "account ID")
	if err != nil {
		return nil, err
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
	exchangeRate := decimal.NewFromInt(1000) // 1 GSALT = 1000 IDR

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Determine if this is an external payment method
		isExternalPayment := paymentMethod != nil && (*paymentMethod == "QRIS" || *paymentMethod == "BANK_TRANSFER" || *paymentMethod == "EWALLET")

		// Set initial status
		status := models.TransactionStatusCompleted
		var completedAt *time.Time
		if !isExternalPayment {
			now := time.Now()
			completedAt = &now
		} else {
			status = models.TransactionStatusPending
		}

		// Create transaction record using helper function
		description := fmt.Sprintf("Topup %d GSALT", amountGsaltUnits/100)
		transaction = s.createBaseTransaction(accountUUID, models.TransactionTypeTopup, amountGsaltUnits, status, &description)

		// Set additional fields
		transaction.ExchangeRateIDR = &exchangeRate
		transaction.CompletedAt = completedAt

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

		// For external payments, create Flip payment request
		if isExternalPayment && s.flipService != nil {
			var flipPaymentMethod models.FlipPaymentMethod = models.FlipPaymentMethodQRIS // Default to QRIS
			if paymentMethod != nil {
				switch *paymentMethod {
				case "QRIS":
					flipPaymentMethod = models.FlipPaymentMethodQRIS
				case "BANK_TRANSFER":
					flipPaymentMethod = models.FlipPaymentMethodVirtualAccount
				case "EWALLET":
					flipPaymentMethod = models.FlipPaymentMethodEWallet
				}
			}

			// Get ConnectUser data directly
			connectUser, err := s.connectService.GetUser(accountUUID.String())
			if err != nil {
				return errors.NewInternalServerError(err, "Failed to get connect user details")
			}

			// Create payment request with details using new method
			transactionIDStr := transaction.ID.String()
			paymentRespWithDetails, err := s.CreateTopupPaymentWithDetails(
				context.Background(),
				transactionIDStr,
				*paymentAmount,
				flipPaymentMethod,
				connectUser,
			)
			if err != nil {
				return errors.NewInternalServerError(err, "Failed to create payment request with Flip")
			}

			// Convert payment response to JSON for storage
			paymentInstructionsJSON, err := s.ConvertPaymentResponseToJSON(&paymentRespWithDetails.FlipPaymentResponse)
			if err != nil {
				return errors.NewInternalServerError(err, "Failed to convert payment instructions to JSON")
			}

			// Update transaction with Flip bill ID and payment instructions
			billID := fmt.Sprintf("%d", paymentRespWithDetails.FlipPaymentResponse.LinkID)
			transaction.ExternalReferenceID = &billID
			transaction.PaymentInstructions = &paymentInstructionsJSON
			if err := tx.Save(transaction).Error; err != nil {
				return errors.NewInternalServerError(err, "Failed to update transaction with Flip payment data")
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

	// Get ConnectUser directly
	connectUser, err := s.connectService.GetUser(request.AccountID)
	if err != nil {
		return models.PaymentResponse{}, fmt.Errorf("connect user not found: %w", err)
	}

	// Calculate fee based on payment method and amount
	fee, err := s.calculateFlipFee(request.PaymentMethod, request.AmountGsaltUnits)
	if err != nil {
		return models.PaymentResponse{}, fmt.Errorf("failed to calculate fee: %w", err)
	}

	// Total amount including fee (user pays the fee)
	totalAmountGsalt := request.AmountGsaltUnits + fee
	totalAmountIDR := totalAmountGsalt * 1000 // Convert to IDR

	// Parse account UUID
	accountUUID, err := s.parseUUID(request.AccountID, "account ID")
	if err != nil {
		return models.PaymentResponse{}, fmt.Errorf("invalid account ID format: %w", err)
	}

	// Create payment transaction using helper function
	description := fmt.Sprintf("Payment via %s", request.PaymentMethod)
	transaction := s.createBaseTransaction(accountUUID, models.TransactionTypePayment, request.AmountGsaltUnits, models.TransactionStatusPending, &description)
	transaction.FeeGsaltUnits = fee
	transaction.TotalAmountGsaltUnits = totalAmountGsalt
	transaction.PaymentMethod = &request.PaymentMethod

	if err := s.db.Create(&transaction).Error; err != nil {
		return models.PaymentResponse{}, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Create payment request data
	transactionIDStr := transaction.ID.String()
	defaultRedirectURL := s.flipService.client.GetDefaultRedirectURL() // Get from config instead of hardcoded

	// Use ConnectUser data for proper customer information
	customerName := connectUser.ID.String()
	customerEmail := connectUser.Email
	if customerEmail == "" {
		customerEmail = fmt.Sprintf("customer-%s@gsalt.com", connectUser.ID.String())
	}
	customerPhone := "081234567890" // Default phone

	paymentData := models.FlipPaymentRequestData{
		TransactionID:         transactionIDStr,
		Title:                 fmt.Sprintf("GSALT Payment - %s", transactionIDStr),
		Amount:                totalAmountIDR,
		ExpiryHours:           24,
		PaymentMethod:         s.mapToFlipPaymentMethod(request.PaymentMethod),
		ReferenceID:           &transactionIDStr,
		ChargeFee:             false,
		IsAddressRequired:     false,
		IsPhoneNumberRequired: true,
		RedirectURL:           &defaultRedirectURL, // Use configured redirect URL

		// Use proper customer information
		CustomerName:  &customerName,
		CustomerEmail: &customerEmail,
		CustomerPhone: &customerPhone,
	}

	// Set specific payment method configurations
	s.setPaymentMethodSpecificData(&paymentData, request)

	// Create bill via Flip API
	billResponse, err := s.flipService.CreateBill(context.Background(), models.CreateBillRequest{
		Title:                 paymentData.Title,
		Type:                  "SINGLE",
		Amount:                paymentData.Amount,
		RedirectURL:           *paymentData.RedirectURL,
		IsAddressRequired:     paymentData.IsAddressRequired,
		IsPhoneNumberRequired: paymentData.IsPhoneNumberRequired,
		Step:                  "3", // Direct to payment confirmation
		SenderName:            *paymentData.CustomerName,
		SenderEmail:           *paymentData.CustomerEmail,
		SenderPhoneNumber:     *paymentData.CustomerPhone,
		SenderAddress:         "Jakarta, Indonesia",
		ReferenceID:           *paymentData.ReferenceID,
		ChargeFee:             paymentData.ChargeFee,
	})
	if err != nil {
		// Rollback transaction on failure
		s.db.Delete(&transaction)
		return models.PaymentResponse{}, fmt.Errorf("failed to create Flip bill: %w", err)
	}

	// Update transaction with external payment ID
	transaction.ExternalPaymentID = &billResponse.LinkURL
	if err := s.db.Save(&transaction).Error; err != nil {
		return models.PaymentResponse{}, fmt.Errorf("failed to update transaction: %w", err)
	}

	paymentMethodStr := ""
	if transaction.PaymentMethod != nil {
		paymentMethodStr = *transaction.PaymentMethod
	}

	// Parse expiry time from response
	var expiryTime *time.Time
	if billResponse.ExpiredDate != "" {
		if parsedTime, err := time.Parse("2006-01-02 15:04:05", billResponse.ExpiredDate); err == nil {
			expiryTime = &parsedTime
		}
	}

	return models.PaymentResponse{
		TransactionID:       transaction.ID.String(),
		Status:              string(transaction.Status),
		PaymentMethod:       paymentMethodStr,
		Amount:              request.AmountGsaltUnits,
		Fee:                 fee,
		TotalAmount:         totalAmountGsalt,
		PaymentURL:          billResponse.LinkURL,
		ExpiryTime:          expiryTime,
		PaymentInstructions: s.getPaymentInstructions(request.PaymentMethod),
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

// calculateFlipFee calculates the fee for Flip payment based on payment method and amount
func (s *TransactionService) calculateFlipFee(paymentMethod string, amountGsalt int64) (int64, error) {
	// Fee structure based on Flip's pricing (in GSALT units)
	// These are example fees - adjust based on actual Flip pricing
	switch paymentMethod {
	case "VA_BCA", "VA_BNI", "VA_BRI", "VA_MANDIRI", "VA_CIMB", "VA_PERMATA", "VA_BSI":
		return 4, nil // 4 GSALT (4000 IDR) for VA
	case "QRIS":
		fee := amountGsalt * 7 / 1000 // 0.7% of amount
		if fee < 2 {
			fee = 2 // Minimum 2 GSALT (2000 IDR)
		}
		return fee, nil
	case "EWALLET_OVO", "EWALLET_DANA", "EWALLET_GOPAY", "EWALLET_LINKAJA", "EWALLET_SHOPEEPAY":
		return 5, nil // 5 GSALT (5000 IDR) for e-wallets
	case "CREDIT_CARD", "DEBIT_CARD":
		fee := amountGsalt * 29 / 1000 // 2.9% of amount
		if fee < 10 {
			fee = 10 // Minimum 10 GSALT (10000 IDR)
		}
		return fee, nil
	case "RETAIL_ALFAMART", "RETAIL_INDOMARET", "RETAIL_CIRCLEK", "RETAIL_LAWSON":
		return 5, nil // 5 GSALT (5000 IDR) for retail outlets
	case "DIRECT_DEBIT":
		return 3, nil // 3 GSALT (3000 IDR) for direct debit
	default:
		return 4, nil // Default fee 4 GSALT
	}
}

// mapToFlipPaymentMethod maps internal payment method to Flip payment method
func (s *TransactionService) mapToFlipPaymentMethod(paymentMethod string) models.FlipPaymentMethod {
	switch {
	case strings.HasPrefix(paymentMethod, "VA_"):
		return models.FlipPaymentMethodVirtualAccount
	case paymentMethod == "QRIS":
		return models.FlipPaymentMethodQRIS
	case strings.HasPrefix(paymentMethod, "EWALLET_"):
		return models.FlipPaymentMethodEWallet
	case strings.HasPrefix(paymentMethod, "CREDIT_CARD"):
		return models.FlipPaymentMethodCreditCard
	case strings.HasPrefix(paymentMethod, "DEBIT_CARD"):
		return models.FlipPaymentMethodDebitCard
	case strings.HasPrefix(paymentMethod, "RETAIL_"):
		return models.FlipPaymentMethodRetailOutlet
	case paymentMethod == "DIRECT_DEBIT":
		return models.FlipPaymentMethodDirectDebit
	case paymentMethod == "BANK_TRANSFER":
		return models.FlipPaymentMethodBankTransfer
	default:
		return models.FlipPaymentMethodQRIS
	}
}

// setPaymentMethodSpecificData sets specific data based on payment method
func (s *TransactionService) setPaymentMethodSpecificData(flipData *models.FlipPaymentRequestData, request models.PaymentRequest) {
	switch {
	case strings.HasPrefix(request.PaymentMethod, "VA_"):
		// Extract bank from payment method (e.g., VA_BCA -> bca)
		bankCode := strings.ToLower(strings.TrimPrefix(request.PaymentMethod, "VA_"))
		flipData.VABank = (*models.FlipVABank)(&bankCode)
	case strings.HasPrefix(request.PaymentMethod, "EWALLET_"):
		// Extract provider from payment method (e.g., EWALLET_OVO -> ovo)
		provider := strings.ToLower(strings.TrimPrefix(request.PaymentMethod, "EWALLET_"))
		flipData.EWalletProvider = (*models.FlipEWalletProvider)(&provider)
	case request.PaymentMethod == "CREDIT_CARD":
		if request.CreditCardProvider != nil {
			flipData.CreditCardProvider = (*models.FlipCreditCardProvider)(request.CreditCardProvider)
		}
	case strings.HasPrefix(request.PaymentMethod, "RETAIL_"):
		// Extract provider from payment method (e.g., RETAIL_ALFAMART -> alfamart)
		provider := strings.ToLower(strings.TrimPrefix(request.PaymentMethod, "RETAIL_"))
		flipData.RetailProvider = (*models.FlipRetailProvider)(&provider)
	case request.PaymentMethod == "DIRECT_DEBIT":
		if request.DirectDebitBank != nil {
			flipData.DirectDebitBank = (*models.FlipDirectDebitBank)(request.DirectDebitBank)
		}
	}
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

// ========== GSALT BUSINESS LOGIC METHODS FOR FLIP INTEGRATION ==========

// CreateTopupPaymentWithDetails creates a payment bill with detailed payment method information
func (s *TransactionService) CreateTopupPaymentWithDetails(ctx context.Context, transactionID string, amountIDR int64, paymentMethod models.FlipPaymentMethod, connectUser *models.ConnectUser) (*models.FlipPaymentResponseWithDetails, error) {
	// Generate internal GSALT reference ID (this should be stored in external_reference_id)
	internalRef := fmt.Sprintf("GSALT-%s", pkg.RandomNumberString(10))

	// Prepare customer information from connectUser
	customerName := connectUser.ID.String()
	customerEmail := connectUser.Email
	if customerEmail == "" {
		customerEmail = fmt.Sprintf("customer-%s@gsalt.com", connectUser.ID.String())
	}
	customerPhone := "081234567890" // Default phone number

	// Create bill request
	billReq := models.CreateBillRequest{
		Title:                 fmt.Sprintf("GSALT Topup - %s", internalRef), // Use internal reference in title
		Type:                  "SINGLE",
		Amount:                amountIDR,
		RedirectURL:           s.flipService.client.GetDefaultRedirectURL(),
		IsAddressRequired:     false,
		IsPhoneNumberRequired: true,
		Step:                  "3", // Direct to payment confirmation
		SenderName:            customerName,
		SenderEmail:           customerEmail,
		SenderPhoneNumber:     customerPhone,
		SenderAddress:         "Jakarta, Indonesia",
		ReferenceID:           internalRef, // Use internal reference for Flip's reference
		ChargeFee:             false,
	}

	// Set payment method specific fields
	s.setPaymentMethodFields(&billReq, paymentMethod)

	// Create bill via Flip API
	billResp, err := s.flipService.CreateBill(ctx, billReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create bill: %w", err)
	}

	// Get complete bill details with payment information
	completeBillResp, err := s.flipService.GetBill(ctx, billResp.LinkID)
	if err != nil {
		// If GetBill fails, continue with basic response but log the error
		fmt.Printf("Warning: Failed to get complete bill details: %v\n", err)
		completeBillResp = nil
	}

	// Update the transaction with correct IDs
	transactionUUID, err := s.parseUUID(transactionID, "transaction ID")
	if err != nil {
		return nil, fmt.Errorf("invalid transaction ID: %w", err)
	}

	// Get the transaction to update it
	var transaction models.Transaction
	if err := s.db.First(&transaction, "id = ?", transactionUUID).Error; err != nil {
		return nil, fmt.Errorf("transaction not found: %w", err)
	}

	// Prepare detailed payment instructions based on response structure
	paymentInstructions := map[string]interface{}{
		"link_id":     billResp.LinkID,
		"link_url":    billResp.LinkURL,
		"title":       billResp.Title,
		"type":        billResp.Type,
		"amount":      billResp.Amount,
		"status":      billResp.Status,
		"step":        billResp.Step,
		"payment_url": billResp.LinkURL,
		"customer": map[string]interface{}{
			"name":    customerName,
			"email":   customerEmail,
			"phone":   customerPhone,
			"address": "Jakarta, Indonesia",
		},
		"payment_method":     string(paymentMethod),
		"internal_reference": internalRef,
		"created_at":         time.Now().Unix(),
		"expired_date":       billResp.ExpiredDate,
		"redirect_url":       billResp.RedirectURL,
	}

	// Add bill_payment details if we have complete response with payment details
	if completeBillResp != nil && len(completeBillResp.BillPayments) > 0 {
		// Use the first (latest) bill payment
		billPayment := completeBillResp.BillPayments[0]
		paymentInstructions["bill_payment"] = map[string]interface{}{
			"id":           billPayment.ID,
			"amount":       billPayment.Amount,
			"status":       billPayment.Status,
			"sender_name":  billPayment.SenderName,
			"sender_email": billPayment.SenderEmail,
			"sender_phone": billPayment.SenderPhoneNumber,
			"created_at":   billPayment.Created.Unix(),
			"updated_at":   billPayment.Updated.Unix(),
		}

		// Add specific payment instructions based on payment method
		switch paymentMethod {
		case models.FlipPaymentMethodVirtualAccount:
			paymentInstructions["instructions"] = "Transfer ke nomor Virtual Account yang disediakan melalui ATM, mobile banking, atau internet banking"
			paymentInstructions["sender_bank"] = billReq.SenderBank
			paymentInstructions["sender_bank_type"] = billReq.SenderBankType
		case models.FlipPaymentMethodQRIS:
			paymentInstructions["qr_code_url"] = billResp.LinkURL
			paymentInstructions["instructions"] = "Scan kode QR menggunakan aplikasi pembayaran digital Anda"
			paymentInstructions["sender_bank"] = "qris"
			paymentInstructions["sender_bank_type"] = "wallet_account"
		case models.FlipPaymentMethodEWallet:
			paymentInstructions["wallet_type"] = billReq.SenderBank
			paymentInstructions["instructions"] = "Bayar menggunakan aplikasi e-wallet yang dipilih"
			paymentInstructions["sender_bank"] = billReq.SenderBank
			paymentInstructions["sender_bank_type"] = "wallet_account"
		case models.FlipPaymentMethodCreditCard:
			paymentInstructions["instructions"] = "Masukkan informasi kartu kredit untuk menyelesaikan pembayaran"
			paymentInstructions["sender_bank"] = "credit_card"
			paymentInstructions["sender_bank_type"] = "credit_card_account"
		case models.FlipPaymentMethodRetailOutlet:
			paymentInstructions["instructions"] = "Kunjungi outlet retail dan berikan kode pembayaran kepada kasir"
			paymentInstructions["sender_bank"] = billReq.SenderBank
			paymentInstructions["sender_bank_type"] = "online_to_offline_account"
		}
	} else {
		// If no detailed payment info available, add basic instructions
		paymentInstructions["sender_bank"] = billReq.SenderBank
		paymentInstructions["sender_bank_type"] = billReq.SenderBankType

		switch paymentMethod {
		case models.FlipPaymentMethodVirtualAccount:
			paymentInstructions["instructions"] = "Transfer ke nomor Virtual Account yang akan disediakan"
		case models.FlipPaymentMethodQRIS:
			paymentInstructions["instructions"] = "Scan kode QR untuk pembayaran"
		case models.FlipPaymentMethodEWallet:
			paymentInstructions["instructions"] = "Bayar menggunakan aplikasi e-wallet"
		default:
			paymentInstructions["instructions"] = "Ikuti instruksi pembayaran di halaman checkout"
		}
	}

	// Convert to JSON and store in PaymentInstructions field
	instructionsJSON, err := json.Marshal(paymentInstructions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment instructions: %w", err)
	}

	// Update transaction with correct field assignments:
	// - external_reference_id = internal GSALT reference
	// - external_payment_id = Flip Link ID for tracking
	// - payment_instructions = complete payment details JSON
	linkIDStr := fmt.Sprintf("%d", billResp.LinkID)
	instructionsJSONStr := string(instructionsJSON)
	transaction.ExternalReferenceID = &internalRef         // GSALT internal reference
	transaction.ExternalPaymentID = &linkIDStr             // Flip's Link ID for tracking
	transaction.PaymentInstructions = &instructionsJSONStr // Complete payment details

	if err := s.db.Save(&transaction).Error; err != nil {
		return nil, fmt.Errorf("failed to update transaction with payment details: %w", err)
	}

	// Convert to our standard response format
	flipPaymentResp := models.FlipPaymentResponse{
		BillID:        0, // Will be set from bill payments
		LinkID:        billResp.LinkID,
		LinkURL:       billResp.LinkURL,
		Title:         billResp.Title,
		Amount:        billResp.Amount,
		Status:        models.FlipStatusPending,
		PaymentMethod: string(paymentMethod),
		CreatedAt:     billResp.Created,
		RedirectURL:   billResp.RedirectURL,
	}

	// Parse expiry date if available
	if billResp.ExpiredDate != "" {
		if expiryTime, err := time.Parse("2006-01-02 15:04:05", billResp.ExpiredDate); err == nil {
			flipPaymentResp.ExpiryTime = &expiryTime
		}
	}

	// Create payment method details
	paymentMethodDetails := s.createPaymentMethodDetails(paymentMethod, billResp)

	return &models.FlipPaymentResponseWithDetails{
		FlipPaymentResponse:  flipPaymentResp,
		PaymentMethodDetails: paymentMethodDetails,
	}, nil
}

// ConvertPaymentResponseToJSON converts payment response to JSON string for storage
func (s *TransactionService) ConvertPaymentResponseToJSON(paymentResp *models.FlipPaymentResponse) (string, error) {
	jsonBytes, err := json.Marshal(paymentResp)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payment response: %w", err)
	}
	return string(jsonBytes), nil
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
	return gsaltUnits * 10 // Since gsaltUnits are in units (100 units = 1 GSALT)
}

// CreateGSALTDisbursement creates a disbursement for GSALT withdrawal
func (s *TransactionService) CreateGSALTDisbursement(ctx context.Context, req models.DisbursementRequest) (*models.DisbursementResponse, error) {
	return s.flipService.CreateDisbursement(ctx, req)
}

// GetSupportedBanksForFlip returns list of supported banks for disbursement from Flip
func (s *TransactionService) GetSupportedBanksForFlip(ctx context.Context) ([]models.FlipBank, error) {
	return s.flipService.GetBanks(ctx)
}

// setPaymentMethodFields sets payment method specific fields for bill creation
func (s *TransactionService) setPaymentMethodFields(billReq *models.CreateBillRequest, paymentMethod models.FlipPaymentMethod) {
	switch paymentMethod {
	case models.FlipPaymentMethodVirtualAccount:
		billReq.SenderBank = "bca" // Default to BCA VA
		billReq.SenderBankType = "virtual_account"
	case models.FlipPaymentMethodQRIS:
		billReq.SenderBank = "qris"
		billReq.SenderBankType = "wallet_account"
	case models.FlipPaymentMethodEWallet:
		billReq.SenderBank = "ovo" // Default to OVO
		billReq.SenderBankType = "wallet_account"
	case models.FlipPaymentMethodCreditCard:
		billReq.SenderBank = "credit_card"
		billReq.SenderBankType = "credit_card_account"
	case models.FlipPaymentMethodDebitCard:
		billReq.SenderBank = "credit_card" // Use same as credit card
		billReq.SenderBankType = "credit_card_account"
	case models.FlipPaymentMethodRetailOutlet:
		billReq.SenderBank = "alfamart"
		billReq.SenderBankType = "online_to_offline_account"
	case models.FlipPaymentMethodDirectDebit:
		billReq.SenderBank = "bca" // Default to BCA
		billReq.SenderBankType = "bank_account"
	case models.FlipPaymentMethodBankTransfer:
		billReq.SenderBank = "bca" // Default to BCA
		billReq.SenderBankType = "bank_account"
	default:
		// Default to BCA Virtual Account as it's most commonly used
		billReq.SenderBank = "bca"
		billReq.SenderBankType = "virtual_account"
	}
}

// setPaymentMethodFieldsWithSpecificBank sets payment method fields with specific bank selection
func (s *TransactionService) setPaymentMethodFieldsWithSpecificBank(billReq *models.CreateBillRequest, paymentMethod models.FlipPaymentMethod, specificBank *string) {
	switch paymentMethod {
	case models.FlipPaymentMethodVirtualAccount:
		if specificBank != nil {
			billReq.SenderBank = *specificBank // Use specified bank
		} else {
			billReq.SenderBank = "bca" // Default to BCA VA
		}
		billReq.SenderBankType = "virtual_account"
	case models.FlipPaymentMethodQRIS:
		billReq.SenderBank = "qris"
		billReq.SenderBankType = "wallet_account"
	case models.FlipPaymentMethodEWallet:
		if specificBank != nil {
			billReq.SenderBank = *specificBank // ovo, shopeepay_app, linkaja, dana
		} else {
			billReq.SenderBank = "ovo" // Default to OVO
		}
		billReq.SenderBankType = "wallet_account"
	case models.FlipPaymentMethodCreditCard:
		billReq.SenderBank = "credit_card"
		billReq.SenderBankType = "credit_card_account"
	case models.FlipPaymentMethodDebitCard:
		billReq.SenderBank = "credit_card"
		billReq.SenderBankType = "credit_card_account"
	case models.FlipPaymentMethodRetailOutlet:
		if specificBank != nil {
			billReq.SenderBank = *specificBank // alfamart, etc.
		} else {
			billReq.SenderBank = "alfamart"
		}
		billReq.SenderBankType = "online_to_offline_account"
	case models.FlipPaymentMethodDirectDebit, models.FlipPaymentMethodBankTransfer:
		if specificBank != nil {
			billReq.SenderBank = *specificBank // bca, mandiri, bni, bri, etc.
		} else {
			billReq.SenderBank = "bca" // Default to BCA
		}
		billReq.SenderBankType = "bank_account"
	default:
		// Default to BCA Virtual Account
		billReq.SenderBank = "bca"
		billReq.SenderBankType = "virtual_account"
	}
}

// createPaymentMethodDetails creates payment method details based on the payment method
func (s *TransactionService) createPaymentMethodDetails(paymentMethod models.FlipPaymentMethod, billResp *models.BillResponse) *models.PaymentMethodDetails {
	details := &models.PaymentMethodDetails{
		LinkID:      billResp.LinkID,
		LinkURL:     billResp.LinkURL,
		Status:      billResp.Status,
		Amount:      billResp.Amount,
		ExpiredDate: billResp.ExpiredDate,
		PaymentURL:  billResp.LinkURL,
	}

	switch paymentMethod {
	case models.FlipPaymentMethodVirtualAccount:
		details.VABank = "bca" // Default
		details.VANumber = ""  // Will be filled by Flip
	case models.FlipPaymentMethodQRIS:
		details.QRCodeURL = ""    // Will be filled by Flip
		details.QRCodeString = "" // Will be filled by Flip
	case models.FlipPaymentMethodEWallet:
		details.EWalletURL = billResp.LinkURL
	}

	return details
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
