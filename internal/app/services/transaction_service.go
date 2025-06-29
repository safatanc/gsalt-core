package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/models"
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

type TransactionService struct {
	db             *gorm.DB
	validator      *infrastructures.Validator
	accountService *AccountService
	flipService    *FlipService
	limits         TransactionLimits
}

func NewTransactionService(db *gorm.DB, validator *infrastructures.Validator, accountService *AccountService, flipService *FlipService) *TransactionService {
	return &TransactionService{
		db:             db,
		validator:      validator,
		accountService: accountService,
		flipService:    flipService,
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

			// Get account info for customer details
			var account models.Account
			if err := tx.Where("connect_id = ?", accountUUID).First(&account).Error; err != nil {
				return errors.NewInternalServerError(err, "Failed to get account details")
			}

			// Create payment request data with V3 features
			transactionIDStr := transaction.ID
			paymentData := models.FlipPaymentRequestData{
				TransactionID:         transactionIDStr,
				Title:                 fmt.Sprintf("GSALT Topup - %s", transactionIDStr),
				Amount:                *paymentAmount,
				ExpiryHours:           24, // 24 hours expiry
				PaymentMethod:         flipPaymentMethod,
				ReferenceID:           &transactionIDStr, // Use transaction ID as reference
				ChargeFee:             false,             // Don't charge additional fee to customer
				IsAddressRequired:     false,             // Address not required for topup
				IsPhoneNumberRequired: true,              // Phone number required for verification
			}

			// Set customer information from account if available
			customerName := account.ConnectID.String()
			paymentData.CustomerName = &customerName

			// Add item details for better payment description
			paymentData.ItemDetails = []models.ItemDetail{
				{
					Name:     fmt.Sprintf("GSALT Topup - %d GSALT", amountGsaltUnits/100),
					Price:    *paymentAmount,
					Quantity: 1,
					Desc:     fmt.Sprintf("Topup %d GSALT to account %s", amountGsaltUnits/100, account.ConnectID.String()),
				},
			}

			// Create payment request with Flip V3
			paymentResp, err := s.flipService.CreateBill(
				context.Background(),
				paymentData,
			)
			if err != nil {
				return errors.NewInternalServerError(err, "Failed to create payment request with Flip")
			}

			// Convert payment response to JSON for storage
			paymentInstructionsJSON, err := s.flipService.ConvertPaymentResponseToJSON(paymentResp)
			if err != nil {
				return errors.NewInternalServerError(err, "Failed to convert payment instructions to JSON")
			}

			// Update transaction with Flip bill ID and payment instructions
			billID := fmt.Sprintf("%d", paymentResp.BillID)
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
			ID:                    uuid.New().String(),
			AccountID:             sourceAccount.ConnectID,
			Type:                  models.TransactionTypeTransferOut,
			AmountGsaltUnits:      amountGsaltUnits,
			TotalAmountGsaltUnits: amountGsaltUnits,
			Currency:              "GSALT",
			Status:                models.TransactionStatusCompleted,
			Description:           description,
			DestinationAccountID:  &destAccount.ConnectID,
			CompletedAt:           &now,
			CreatedAt:             now,
			UpdatedAt:             now,
		}

		if err := tx.Create(transferOut).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create transfer out transaction")
		}

		// Create transfer in transaction
		transferOutID, _ := uuid.Parse(transferOut.ID)
		transferIn = &models.Transaction{
			ID:                    uuid.New().String(),
			AccountID:             destAccount.ConnectID,
			Type:                  models.TransactionTypeTransferIn,
			AmountGsaltUnits:      amountGsaltUnits,
			TotalAmountGsaltUnits: amountGsaltUnits,
			Currency:              "GSALT",
			Status:                models.TransactionStatusCompleted,
			Description:           description,
			SourceAccountID:       &sourceAccount.ConnectID,
			RelatedTransactionID:  &transferOutID,
			CompletedAt:           &now,
			CreatedAt:             now,
			UpdatedAt:             now,
		}

		if err := tx.Create(transferIn).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create transfer in transaction")
		}

		// Update related transaction ID
		transferInID, _ := uuid.Parse(transferIn.ID)
		transferOut.RelatedTransactionID = &transferInID
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

// ProcessPayment processes external payment (QRIS, Bank Transfer, E-wallet, Credit Card, etc.)
func (s *TransactionService) ProcessPayment(request models.PaymentRequest) (models.PaymentResponse, error) {
	// Validate request
	if err := s.validator.Validate(request); err != nil {
		return models.PaymentResponse{}, err
	}

	// Get account to validate
	account, err := s.accountService.GetAccount(request.AccountID)
	if err != nil {
		return models.PaymentResponse{}, fmt.Errorf("account not found: %w", err)
	}

	// Calculate fee based on payment method and amount
	fee, err := s.calculateFlipFee(request.PaymentMethod, request.AmountGsaltUnits)
	if err != nil {
		return models.PaymentResponse{}, fmt.Errorf("failed to calculate fee: %w", err)
	}

	// Total amount including fee (user pays the fee)
	totalAmountGsalt := request.AmountGsaltUnits + fee
	totalAmountIDR := totalAmountGsalt * 1000 // Convert to IDR

	// Create transaction record first
	transaction := models.Transaction{
		ID:                    generateTransactionID(),
		AccountID:             account.ConnectID,
		Type:                  models.TransactionTypeTopup,
		Status:                models.TransactionStatusPending,
		AmountGsaltUnits:      request.AmountGsaltUnits,
		FeeGsaltUnits:         fee,
		TotalAmountGsaltUnits: totalAmountGsalt,
		PaymentMethod:         &request.PaymentMethod,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if err := s.db.Create(&transaction).Error; err != nil {
		return models.PaymentResponse{}, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Create Flip payment bill with comprehensive data
	flipData := models.FlipPaymentRequestData{
		TransactionID:   transaction.ID,
		Title:           fmt.Sprintf("GSALT Topup - %d GSALT (+ %d GSALT fee)", request.AmountGsaltUnits, fee),
		Amount:          totalAmountIDR,
		ExpiryHours:     24,
		PaymentMethod:   s.mapToFlipPaymentMethod(request.PaymentMethod),
		CustomerName:    request.CustomerName,
		CustomerEmail:   request.CustomerEmail,
		CustomerPhone:   request.CustomerPhone,
		CustomerAddress: request.CustomerAddress,
		RedirectURL:     request.RedirectURL,
		ReferenceID:     &transaction.ID,
		ChargeFee:       true, // Fee charged to user
		ItemDetails: []models.ItemDetail{
			{
				ID:       "gsalt_topup",
				Name:     "GSALT Token Topup",
				Price:    request.AmountGsaltUnits * 1000,
				Quantity: 1,
				Desc:     fmt.Sprintf("Purchase of %d GSALT tokens", request.AmountGsaltUnits),
			},
			{
				ID:       "flip_fee",
				Name:     "Payment Processing Fee",
				Price:    fee * 1000,
				Quantity: 1,
				Desc:     "Flip payment processing fee",
			},
		},
		IsAddressRequired:     request.CustomerAddress != nil,
		IsPhoneNumberRequired: request.CustomerPhone != nil,
	}

	// Set specific payment method configurations
	s.setPaymentMethodSpecificData(&flipData, request)

	// Create bill via Flip API
	billResponse, err := s.flipService.CreateBill(context.Background(), flipData)
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

	return models.PaymentResponse{
		TransactionID:       transaction.ID,
		Status:              string(transaction.Status),
		PaymentMethod:       paymentMethodStr,
		Amount:              request.AmountGsaltUnits,
		Fee:                 fee,
		TotalAmount:         totalAmountGsalt,
		PaymentURL:          billResponse.LinkURL,
		ExpiryTime:          billResponse.ExpiryTime,
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
			ID:                    uuid.New().String(),
			AccountID:             sourceAccount.ConnectID,
			Type:                  models.TransactionTypeGiftOut,
			AmountGsaltUnits:      amountGsaltUnits,
			TotalAmountGsaltUnits: amountGsaltUnits,
			Currency:              "GSALT",
			Status:                models.TransactionStatusCompleted,
			Description:           description,
			DestinationAccountID:  &destAccount.ConnectID,
			CompletedAt:           &now,
			CreatedAt:             now,
			UpdatedAt:             now,
		}

		if err := tx.Create(giftOut).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create gift out transaction")
		}

		// Create gift in transaction
		giftOutID, _ := uuid.Parse(giftOut.ID)
		giftIn = &models.Transaction{
			ID:                    uuid.New().String(),
			AccountID:             destAccount.ConnectID,
			Type:                  models.TransactionTypeGiftIn,
			AmountGsaltUnits:      amountGsaltUnits,
			TotalAmountGsaltUnits: amountGsaltUnits,
			Currency:              "GSALT",
			Status:                models.TransactionStatusCompleted,
			Description:           description,
			SourceAccountID:       &sourceAccount.ConnectID,
			RelatedTransactionID:  &giftOutID,
			CompletedAt:           &now,
			CreatedAt:             now,
			UpdatedAt:             now,
		}

		if err := tx.Create(giftIn).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to create gift in transaction")
		}

		// Update related transaction ID
		giftInID, _ := uuid.Parse(giftIn.ID)
		giftOut.RelatedTransactionID = &giftInID
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

// ProcessWithdrawal processes GSALT withdrawal to bank account via Flip disbursement
func (s *TransactionService) ProcessWithdrawal(accountId string, amountGsaltUnits int64, bankCode, accountNumber, recipientName string, description *string, externalRefId *string) (*models.Transaction, error) {
	// Parse account ID
	accountUUID, err := uuid.Parse(accountId)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	// Validate withdrawal amount
	if err := s.validateTransactionAmount(models.TransactionTypeWithdrawal, amountGsaltUnits); err != nil {
		return nil, err
	}

	// Check account balance
	var account models.Account
	if err := s.db.First(&account, "connect_id = ?", accountUUID).Error; err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}

	if !s.hasSufficientBalance(account.Balance, amountGsaltUnits) {
		return nil, fmt.Errorf("insufficient balance")
	}

	// Check daily limits
	if err := s.checkDailyLimits(accountUUID, models.TransactionTypeWithdrawal, amountGsaltUnits); err != nil {
		return nil, err
	}

	// Check for idempotency if external reference ID is provided
	if externalRefId != nil {
		existingTxn, err := s.checkIdempotency(externalRefId)
		if err != nil {
			return nil, err
		}
		if existingTxn != nil {
			return existingTxn, nil
		}
	}

	// Validate bank account first
	ctx := context.Background()
	bankInquiry, err := s.flipService.ValidateBankAccount(ctx, accountNumber, bankCode)
	if err != nil {
		return nil, fmt.Errorf("failed to validate bank account: %w", err)
	}

	// Check if account name matches (optional validation)
	if bankInquiry.Status != "SUCCESS" {
		return nil, fmt.Errorf("invalid bank account: %s", bankInquiry.Status)
	}

	// Check disbursement service availability
	available, err := s.flipService.CheckDisbursementAvailability(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check disbursement availability: %w", err)
	}
	if !available {
		return nil, fmt.Errorf("disbursement service is currently unavailable due to maintenance")
	}

	// Begin database transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create withdrawal transaction
	transactionID := uuid.New().String()
	withdrawalDesc := fmt.Sprintf("Withdrawal to %s - %s (%s)", bankCode, accountNumber, recipientName)
	if description != nil && *description != "" {
		withdrawalDesc = fmt.Sprintf("%s - %s", withdrawalDesc, *description)
	}

	transaction := models.Transaction{
		ID:                    transactionID,
		AccountID:             accountUUID,
		Type:                  models.TransactionTypeWithdrawal,
		AmountGsaltUnits:      amountGsaltUnits,
		TotalAmountGsaltUnits: amountGsaltUnits,
		Description:           &withdrawalDesc,
		Status:                models.TransactionStatusPending,
		ExternalReferenceID:   externalRefId,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create withdrawal transaction: %w", err)
	}

	// Update account balance (deduct withdrawal amount)
	if err := s.updateAccountBalance(tx, accountUUID, -amountGsaltUnits); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update account balance: %w", err)
	}

	// Commit database transaction first
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Convert GSALT to IDR for disbursement
	amountIDR := s.flipService.ConvertGSALTToIDR(amountGsaltUnits)

	// Create disbursement request
	disbursementReq := models.GSALTDisbursementRequest{
		TransactionID:  transactionID,
		RecipientName:  recipientName,
		AccountNumber:  accountNumber,
		BankCode:       bankCode,
		AmountGSALT:    amountGsaltUnits,
		AmountIDR:      amountIDR,
		Description:    withdrawalDesc,
		IdempotencyKey: transactionID,
	}

	// Create disbursement via Flip
	disbursementResp, err := s.flipService.CreateGSALTDisbursement(ctx, disbursementReq)
	if err != nil {
		// If disbursement fails, we need to reverse the balance deduction
		reverseTx := s.db.Begin()
		s.updateAccountBalance(reverseTx, accountUUID, amountGsaltUnits) // Add back the amount

		// Update transaction status to failed
		s.db.Model(&transaction).Updates(models.Transaction{
			Status:    models.TransactionStatusFailed,
			UpdatedAt: time.Now(),
		})

		reverseTx.Commit()
		return nil, fmt.Errorf("failed to create disbursement: %w", err)
	}

	// Update transaction with disbursement information
	updateData := models.Transaction{
		Status:    models.TransactionStatusProcessing,
		UpdatedAt: time.Now(),
	}

	// Store disbursement ID as external payment ID
	disbursementIDStr := fmt.Sprintf("%d", disbursementResp.DisbursementID)
	updateData.ExternalPaymentID = &disbursementIDStr

	if err := s.db.Model(&transaction).Updates(updateData).Error; err != nil {
		return nil, fmt.Errorf("failed to update transaction with disbursement info: %w", err)
	}

	// Refresh transaction data
	if err := s.db.First(&transaction, "id = ?", transactionID).Error; err != nil {
		return nil, fmt.Errorf("failed to refresh transaction: %w", err)
	}

	return &transaction, nil
}

// GetSupportedBanksForWithdrawal retrieves list of banks that support withdrawal
func (s *TransactionService) GetSupportedBanksForWithdrawal(ctx context.Context) ([]models.BankListResponse, error) {
	// Get banks from Flip
	flipBanks, err := s.flipService.GetSupportedBanks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get supported banks: %w", err)
	}

	// Convert to our response format
	var bankList []models.BankListResponse
	for _, bank := range flipBanks {
		bankResp := models.BankListResponse{
			BankCode:    bank.BankCode,
			BankName:    bank.Name,
			Available:   bank.Status == "OPERATIONAL",
			MinAmount:   bank.MinAmount,
			MaxAmount:   bank.MaxAmount,
			Fee:         bank.Fee,
			Maintenance: bank.Status != "OPERATIONAL",
		}
		bankList = append(bankList, bankResp)
	}

	return bankList, nil
}

// CheckWithdrawalStatus checks the status of a withdrawal transaction
func (s *TransactionService) CheckWithdrawalStatus(transactionId string) (*models.WithdrawalResponse, error) {
	// Get transaction from database
	transaction, err := s.GetTransaction(transactionId)
	if err != nil {
		return nil, err
	}

	if transaction.Type != models.TransactionTypeWithdrawal {
		return nil, fmt.Errorf("transaction is not a withdrawal")
	}

	response := &models.WithdrawalResponse{
		Transaction:   transaction,
		EstimatedTime: "1-3 business days",
		Status:        string(transaction.Status),
	}

	// If transaction has external payment ID (disbursement ID), add it to response
	if transaction.ExternalPaymentID != nil {
		response.DisbursementID = transaction.ExternalPaymentID
	}

	// Get more detailed status from Flip if available
	if transaction.ExternalPaymentID != nil {
		ctx := context.Background()
		disbursementStatus, err := s.flipService.GetGSALTDisbursementStatus(ctx, transactionId)
		if err == nil {
			response.Status = string(disbursementStatus.Status)
			// Update estimated time based on status
			switch disbursementStatus.Status {
			case models.DisbursementStatusPending:
				response.EstimatedTime = "Processing - 1-3 business days"
			case models.DisbursementStatusProcessed:
				response.EstimatedTime = "In progress - within 24 hours"
			case models.DisbursementStatusDone:
				response.EstimatedTime = "Completed"
			case models.DisbursementStatusCancelled:
				response.EstimatedTime = "Cancelled"
			}
		}
	}

	return response, nil
}

// ValidateBankAccountForWithdrawal validates bank account before withdrawal
func (s *TransactionService) ValidateBankAccountForWithdrawal(ctx context.Context, bankCode, accountNumber string) (*models.BankAccountInquiryResponse, error) {
	return s.flipService.ValidateBankAccount(ctx, accountNumber, bankCode)
}

// GetWithdrawalBalance gets available balance for withdrawal (considering pending withdrawals)
func (s *TransactionService) GetWithdrawalBalance(accountId string) (int64, error) {
	accountUUID, err := uuid.Parse(accountId)
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

// generateTransactionID generates a unique transaction ID
func generateTransactionID() string {
	return fmt.Sprintf("TXN_%d_%s", time.Now().Unix(), uuid.New().String()[:8])
}
