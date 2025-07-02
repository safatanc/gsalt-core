package deliveries

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/middlewares"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/app/pkg"
	"github.com/safatanc/gsalt-core/internal/app/services"
	"github.com/shopspring/decimal"
)

type TransactionHandler struct {
	transactionService   *services.TransactionService
	paymentMethodService *services.PaymentMethodService
	authMiddleware       *middlewares.AuthMiddleware
}

func NewTransactionHandler(transactionService *services.TransactionService, paymentMethodService *services.PaymentMethodService, authMiddleware *middlewares.AuthMiddleware) *TransactionHandler {
	return &TransactionHandler{
		transactionService:   transactionService,
		paymentMethodService: paymentMethodService,
		authMiddleware:       authMiddleware,
	}
}

func (h *TransactionHandler) RegisterRoutes(router fiber.Router) {
	transactionGroup := router.Group("/transactions")
	// Public routes (no auth required)
	transactionGroup.Post("/webhook/flip", h.HandleFlipWebhook)

	// Protected routes (auth required)
	auth := transactionGroup.Group("/", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount)

	// Transaction CRUD
	auth.Post("/", h.CreateTransaction)
	auth.Get("/ref/:ref", h.GetTransactionByRef)
	auth.Get("/:id", h.GetTransaction)
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
}

func (h *TransactionHandler) CreateTransaction(c *fiber.Ctx) error {
	var req models.TransactionCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	transaction, err := h.transactionService.CreateTransaction(&req)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, transaction)
}

func (h *TransactionHandler) GetTransaction(c *fiber.Ctx) error {
	id := c.Params("id")

	transaction, err := h.transactionService.GetTransaction(id)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, transaction)
}

func (h *TransactionHandler) GetTransactionByRef(c *fiber.Ctx) error {
	ref := c.Params("ref")

	transaction, err := h.transactionService.GetTransactionByRef(ref)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, transaction)
}

func (h *TransactionHandler) GetMyTransactions(c *fiber.Ctx) error {
	account := c.Locals("account").(*models.Account)

	// Parse pagination from query parameters
	var pagination models.PaginationRequest

	// Parse page parameter
	pageStr := c.Query("page", "1")
	if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
		pagination.Page = page
	} else {
		pagination.Page = 1
	}

	// Parse limit parameter
	limitStr := c.Query("limit", "10")
	if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
		pagination.Limit = limit
	} else {
		pagination.Limit = 10
	}

	// Parse order parameter
	order := c.Query("order", "desc")
	if order == "asc" || order == "desc" {
		pagination.Order = order
	} else {
		pagination.Order = "desc"
	}

	// Parse order_field parameter
	orderField := c.Query("order_field", "created_at")
	pagination.OrderField = orderField

	transactions, err := h.transactionService.GetTransactionsByAccount(account.ConnectID.String(), &pagination)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, transactions)
}

func (h *TransactionHandler) UpdateTransaction(c *fiber.Ctx) error {
	id := c.Params("id")

	var req models.TransactionUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	transaction, err := h.transactionService.UpdateTransaction(id, &req)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, transaction)
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
		req.PaymentAmount,
		req.PaymentCurrency,
		req.PaymentMethod,
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

func (h *TransactionHandler) ProcessPayment(c *fiber.Ctx) error {
	var req models.PaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	// Get account from context
	account := c.Locals("account").(*models.Account)
	if account == nil {
		return pkg.ErrorResponse(c, errors.NewUnauthorizedError("Account not found in context"))
	}
	req.AccountID = account.ConnectID.String()

	// Process payment
	response, err := h.transactionService.ProcessPayment(req)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, response)
}

func (h *TransactionHandler) ConfirmPayment(c *fiber.Ctx) error {
	id := c.Params("id")

	var req models.ConfirmPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	transaction, err := h.transactionService.ConfirmPayment(id, req.ExternalPaymentID)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, transaction)
}

func (h *TransactionHandler) RejectPayment(c *fiber.Ctx) error {
	id := c.Params("id")

	var req models.RejectPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	transaction, err := h.transactionService.RejectPayment(id, req.Reason)
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

	// TODO: Verify webhook signature
	// callbackToken := c.Get("X-CALLBACK-TOKEN")
	// if !verifyFlipSignature(callbackToken, payload) {
	//     return pkg.ErrorResponse(c, errors.NewUnauthorizedError("Invalid webhook signature"))
	// }

	// Process webhook based on payment status
	// For Flip, we need to extract transaction ID from bill_title or use bill_link_id
	transactionID := payload.BillTitle // Assuming we store transaction ID in bill title
	billID := fmt.Sprintf("%d", payload.BillLinkID)

	switch payload.Status {
	case models.FlipStatusSuccessful:
		// Confirm the payment
		_, err := h.transactionService.ConfirmPayment(transactionID, &billID)
		if err != nil {
			// Log error but don't return error to Flip to prevent retries
			// In production, you might want to use a proper logger
			fmt.Printf("Failed to confirm payment for transaction %s: %v\n", transactionID, err)
		}
	case models.FlipStatusFailed, models.FlipStatusExpired, models.FlipStatusCancelled:
		// Reject the payment
		reason := fmt.Sprintf("Payment %s in Flip", payload.Status)
		_, err := h.transactionService.RejectPayment(transactionID, &reason)
		if err != nil {
			// Log error but don't return error to Flip to prevent retries
			fmt.Printf("Failed to reject payment for transaction %s: %v\n", transactionID, err)
		}
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
