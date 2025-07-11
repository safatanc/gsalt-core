package deliveries

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/middlewares"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/app/pkg"
	"github.com/safatanc/gsalt-core/internal/app/services"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TransactionHandler struct {
	transactionService   *services.TransactionService
	paymentService       *services.PaymentService
	paymentMethodService *services.PaymentMethodService
	authMiddleware       *middlewares.AuthMiddleware
}

func NewTransactionHandler(
	transactionService *services.TransactionService,
	paymentService *services.PaymentService,
	paymentMethodService *services.PaymentMethodService,
	authMiddleware *middlewares.AuthMiddleware,
) *TransactionHandler {
	return &TransactionHandler{
		transactionService:   transactionService,
		paymentService:       paymentService,
		paymentMethodService: paymentMethodService,
		authMiddleware:       authMiddleware,
	}
}

func (h *TransactionHandler) RegisterRoutes(router fiber.Router) {
	transactionGroup := router.Group("/transactions")

	// Public routes (no auth required)
	transactionGroup.Post("/webhook/flip", h.HandleFlipWebhook)
	transactionGroup.Get("/ref/:ref", h.GetTransactionByRef)

	// Protected routes (auth required)
	auth := transactionGroup.Group("/", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount)

	// Transaction CRUD
	auth.Post("/", h.CreateTransaction)
	auth.Get("/", h.GetMyTransactions)
	auth.Put("/:id", h.UpdateTransaction)

	// Transaction operations
	auth.Post("/topup", h.ProcessTopup)
	auth.Post("/transfer", h.ProcessTransfer)
	auth.Post("/payment", h.ProcessPayment)
	auth.Get("/payment-methods", h.GetSupportedPaymentMethods)
	auth.Post("/:id/confirm", h.ConfirmPayment)
	auth.Post("/:id/reject", h.RejectPayment)

	// Withdrawal operations
	auth.Post("/withdrawal", h.ProcessWithdrawal)
	auth.Get("/withdrawal/banks", h.GetSupportedBanksForWithdrawal)
	auth.Get("/withdrawal/balance", h.GetWithdrawalBalance)
	auth.Post("/withdrawal/:id/status", h.CheckWithdrawalStatus)
	auth.Post("/withdrawal/validate-bank-account", h.ValidateBankAccountForWithdrawal)

	// Move the :id route to the bottom to avoid catching other routes
	transactionGroup.Get("/:id", h.GetTransaction)
}

// CreateTransaction handles transaction creation
func (h *TransactionHandler) CreateTransaction(c *fiber.Ctx) error {
	var req models.TransactionCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, errors.NewBadRequestError("Invalid request body"))
	}

	// Get account from context
	account := c.Locals("account").(*models.Account)
	if account == nil {
		return pkg.ErrorResponse(c, errors.NewUnauthorizedError("Account not found in context"))
	}
	req.AccountID = account.ConnectID.String()

	// Create transaction
	transaction, err := h.transactionService.CreateTransaction(&req)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, &models.TransactionResponse{
		Transaction: transaction,
	})
}

// UpdateTransaction handles transaction update
func (h *TransactionHandler) UpdateTransaction(c *fiber.Ctx) error {
	id := c.Params("id")
	transactionID, err := uuid.Parse(id)
	if err != nil {
		return pkg.ErrorResponse(c, errors.NewBadRequestError("Invalid transaction ID"))
	}

	var req models.TransactionUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, errors.NewBadRequestError("Invalid request body"))
	}

	// Update transaction
	transaction, err := h.transactionService.UpdateTransaction(transactionID.String(), &req)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, &models.TransactionResponse{
		Transaction: transaction,
	})
}

// GetTransaction retrieves a transaction by ID
func (h *TransactionHandler) GetTransaction(c *fiber.Ctx) error {
	id := c.Params("id")
	transactionID, err := uuid.Parse(id)
	if err != nil {
		return pkg.ErrorResponse(c, errors.NewBadRequestError("Invalid transaction ID"))
	}

	transaction, err := h.transactionService.GetTransaction(transactionID.String())
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	// Get payment details if exists
	details, err := h.paymentService.GetPaymentDetailsByTransactionID(c.Context(), transaction.ID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, &models.TransactionResponse{
		Transaction:    transaction,
		PaymentDetails: details,
	})
}

// GetTransactionByRef retrieves a transaction by external reference ID
func (h *TransactionHandler) GetTransactionByRef(c *fiber.Ctx) error {
	ref := c.Params("ref")
	transaction, err := h.transactionService.GetTransactionByRef(ref)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, transaction)
}

// GetMyTransactions retrieves transactions for the authenticated user
func (h *TransactionHandler) GetMyTransactions(c *fiber.Ctx) error {
	// Get account from context
	account := c.Locals("account").(*models.Account)
	if account == nil {
		return pkg.ErrorResponse(c, errors.NewUnauthorizedError("Account not found in context"))
	}

	// Get pagination params
	pagination := &models.PaginationRequest{
		Page:  c.QueryInt("page", 1),
		Limit: c.QueryInt("limit", 10),
	}

	// Get transactions
	result, err := h.transactionService.GetTransactionsByAccount(account.ConnectID.String(), pagination)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	// Get payment details for each transaction
	responses := make([]*models.TransactionResponse, 0)
	for _, transaction := range result.Items {
		details, err := h.paymentService.GetPaymentDetailsByTransactionID(c.Context(), transaction.ID)
		if err != nil && err != gorm.ErrRecordNotFound {
			return pkg.ErrorResponse(c, err)
		}

		responses = append(responses, &models.TransactionResponse{
			Transaction:    &transaction,
			PaymentDetails: details,
		})
	}

	return pkg.SuccessResponse(c, &models.Pagination[[]*models.TransactionResponse]{
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
		TotalItems: result.TotalItems,
		HasNext:    result.HasNext,
		HasPrev:    result.HasPrev,
		Items:      responses,
	})
}

func (h *TransactionHandler) ProcessTopup(c *fiber.Ctx) error {
	account := c.Locals("account").(*models.Account)

	var req models.TopupRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	// Convert GSALT amount to units (1 GSALT = 100 units)
	amountGsalt, err := decimal.NewFromString(req.AmountGsalt)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}
	amountGsaltUnits := amountGsalt.Mul(decimal.NewFromInt(100)).IntPart()

	topupResponse, err := h.transactionService.ProcessTopup(
		account.ConnectID.String(),
		amountGsaltUnits,
		*req.PaymentMethod,
		req.ExternalReferenceID,
	)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, topupResponse)
}

func (h *TransactionHandler) ProcessTransfer(c *fiber.Ctx) error {
	account := c.Locals("account").(*models.Account)

	var req models.TransferRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	// Convert GSALT amount to units (1 GSALT = 100 units)
	amountGsalt, err := decimal.NewFromString(req.AmountGsalt)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}
	amountGsaltUnits := amountGsalt.Mul(decimal.NewFromInt(100)).IntPart()

	transferOut, transferIn, err := h.transactionService.ProcessTransfer(
		account.ConnectID.String(),
		req.DestinationAccountID,
		amountGsaltUnits,
		req.Description,
	)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	response := map[string]interface{}{
		"transfer_out": transferOut,
		"transfer_in":  transferIn,
	}

	return pkg.SuccessResponse(c, response)
}

// ProcessPayment handles payment transaction
func (h *TransactionHandler) ProcessPayment(c *fiber.Ctx) error {
	var req models.PaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, errors.NewBadRequestError("Invalid request body"))
	}

	// Get account from context
	account := c.Locals("account").(*models.Account)
	if account == nil {
		return pkg.ErrorResponse(c, errors.NewUnauthorizedError("Account not found in context"))
	}
	req.AccountID = account.ConnectID.String()

	// Process payment
	transaction, err := h.transactionService.ProcessPayment(req)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, transaction)
}

// ConfirmPayment confirms a payment transaction
func (h *TransactionHandler) ConfirmPayment(c *fiber.Ctx) error {
	id := c.Params("id")
	transactionID, err := uuid.Parse(id)
	if err != nil {
		return pkg.ErrorResponse(c, errors.NewBadRequestError("Invalid transaction ID"))
	}

	var req models.PaymentStatusUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, errors.NewBadRequestError("Invalid request body"))
	}

	// Set status to completed
	req.Status = models.PaymentStatusCompleted
	now := time.Now()
	req.PaymentTime = &now

	// Update payment status
	err = h.paymentService.UpdatePaymentStatus(c.Context(), transactionID, &req)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	// Get updated transaction
	transaction, err := h.transactionService.GetTransaction(id)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, transaction)
}

// RejectPayment rejects a payment transaction
func (h *TransactionHandler) RejectPayment(c *fiber.Ctx) error {
	id := c.Params("id")
	transactionID, err := uuid.Parse(id)
	if err != nil {
		return pkg.ErrorResponse(c, errors.NewBadRequestError("Invalid transaction ID"))
	}

	var req models.PaymentStatusUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, errors.NewBadRequestError("Invalid request body"))
	}

	// Set status to failed
	req.Status = models.PaymentStatusFailed

	// Update payment status
	err = h.paymentService.UpdatePaymentStatus(c.Context(), transactionID, &req)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	// Get updated transaction
	transaction, err := h.transactionService.GetTransaction(id)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, transaction)
}

// HandleFlipWebhook handles webhook notifications from Flip
func (h *TransactionHandler) HandleFlipWebhook(c *fiber.Ctx) error {
	var payload models.FlipWebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		return pkg.ErrorResponse(c, errors.NewBadRequestError("Invalid webhook payload"))
	}

	// Process webhook based on payment status
	transactionID := payload.BillTitle // Assuming we store transaction ID in bill title
	txID, err := uuid.Parse(transactionID)
	if err != nil {
		return pkg.ErrorResponse(c, errors.NewBadRequestError("Invalid transaction ID in bill title"))
	}

	billID := fmt.Sprintf("%d", payload.BillLinkID)
	now := time.Now()

	var req models.PaymentStatusUpdateRequest
	req.ProviderPaymentID = &billID
	req.PaymentTime = &now

	switch payload.Status {
	case models.FlipStatusSuccessful:
		req.Status = models.PaymentStatusCompleted
	case models.FlipStatusFailed:
		req.Status = models.PaymentStatusFailed
		desc := fmt.Sprintf("Payment failed in Flip: %s", payload.Status)
		req.StatusDescription = &desc
	case models.FlipStatusExpired:
		req.Status = models.PaymentStatusExpired
		desc := "Payment expired in Flip"
		req.StatusDescription = &desc
	case models.FlipStatusCancelled:
		req.Status = models.PaymentStatusCancelled
		desc := "Payment cancelled in Flip"
		req.StatusDescription = &desc
	default:
		return pkg.ErrorResponse(c, errors.NewBadRequestError("Invalid payment status"))
	}

	// Update payment status
	err = h.paymentService.UpdatePaymentStatus(c.Context(), txID, &req)
	if err != nil {
		// Log error but don't return error to Flip to prevent retries
		fmt.Printf("Failed to update payment status for transaction %s: %v\n", transactionID, err)
	}

	// Always return success to Flip to prevent retries
	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Webhook processed",
	})
}

// GetSupportedPaymentMethods returns all supported payment methods with their fees
func (h *TransactionHandler) GetSupportedPaymentMethods(c *fiber.Ctx) error {
	// 1. Bind filter from query parameters
	var filter models.PaymentMethodFilter
	if err := c.QueryParser(&filter); err != nil {
		return pkg.ErrorResponse(c, errors.NewBadRequestError("Invalid query parameters"))
	}

	// 2. Call the service to get the methods
	methods, err := h.paymentMethodService.GetPaymentMethods(&filter)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, methods)
}

// ProcessWithdrawal handles withdrawal processing
func (h *TransactionHandler) ProcessWithdrawal(c *fiber.Ctx) error {
	account := c.Locals("account").(*models.Account)

	var req models.WithdrawalRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	// Convert amount from string to GSALT units
	amountGsalt, err := decimal.NewFromString(req.AmountGsalt)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}
	amountGsaltUnits := amountGsalt.Mul(decimal.NewFromInt(100)).IntPart()

	transaction, err := h.transactionService.ProcessWithdrawal(
		account.ConnectID.String(),
		amountGsaltUnits,
		req.BankCode,
		req.AccountNumber,
		req.RecipientName,
		req.Description,
		req.ExternalReferenceID,
	)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, transaction)
}

// GetSupportedBanksForWithdrawal returns list of banks that support withdrawal
func (h *TransactionHandler) GetSupportedBanksForWithdrawal(c *fiber.Ctx) error {
	ctx := c.Context()
	banks, err := h.transactionService.GetSupportedBanksForWithdrawal(ctx)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, banks)
}

// GetWithdrawalBalance returns available balance for withdrawal
func (h *TransactionHandler) GetWithdrawalBalance(c *fiber.Ctx) error {
	account := c.Locals("account").(*models.Account)

	balance, err := h.transactionService.GetWithdrawalBalance(account.ConnectID.String())
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	// Convert balance to GSALT decimal for display
	balanceGsalt := decimal.NewFromInt(balance).Div(decimal.NewFromInt(100))

	response := map[string]interface{}{
		"balance_gsalt_units": balance,
		"balance_gsalt":       balanceGsalt,
		"balance_idr":         balanceGsalt.Mul(decimal.NewFromInt(1000)),
	}

	return pkg.SuccessResponse(c, response)
}

// CheckWithdrawalStatus checks the status of a withdrawal transaction
func (h *TransactionHandler) CheckWithdrawalStatus(c *fiber.Ctx) error {
	transactionId := c.Params("id")

	response, err := h.transactionService.CheckWithdrawalStatus(transactionId)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, response)
}

// ValidateBankAccountForWithdrawal validates bank account before withdrawal
func (h *TransactionHandler) ValidateBankAccountForWithdrawal(c *fiber.Ctx) error {
	var req models.BankAccountInquiryRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	ctx := c.Context()
	response, err := h.transactionService.ValidateBankAccountForWithdrawal(ctx, req.BankCode, req.AccountNumber)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, response)
}
