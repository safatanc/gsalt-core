package deliveries

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/safatanc/gsalt-core/internal/app/middlewares"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/app/pkg"
	"github.com/safatanc/gsalt-core/internal/app/services"
	"github.com/shopspring/decimal"
)

type TransactionHandler struct {
	transactionService *services.TransactionService
	authMiddleware     *middlewares.AuthMiddleware
}

func NewTransactionHandler(transactionService *services.TransactionService, authMiddleware *middlewares.AuthMiddleware) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
		authMiddleware:     authMiddleware,
	}
}

func (h *TransactionHandler) RegisterRoutes(router fiber.Router) {
	transactionGroup := router.Group("/transactions")

	transactionGroup.Post("/", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.CreateTransaction)
	transactionGroup.Get("/me", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.GetMyTransactions)
	transactionGroup.Get("/:id", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.GetTransaction)
	transactionGroup.Patch("/:id", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.UpdateTransaction)

	// Payment operations
	transactionGroup.Post("/topup", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.ProcessTopup)
	transactionGroup.Post("/transfer", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.ProcessTransfer)
	transactionGroup.Post("/payment", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.ProcessPayment)
}

func (h *TransactionHandler) CreateTransaction(c *fiber.Ctx) error {
	var dto models.TransactionCreateDto
	if err := c.BodyParser(&dto); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	transaction, err := h.transactionService.CreateTransaction(&dto)
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

	var dto models.TransactionUpdateDto
	if err := c.BodyParser(&dto); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	transaction, err := h.transactionService.UpdateTransaction(id, &dto)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, transaction)
}

type TopupRequest struct {
	AmountGsalt         string  `json:"amount_gsalt" validate:"required"`
	PaymentAmount       *int64  `json:"payment_amount,omitempty"`
	PaymentCurrency     *string `json:"payment_currency,omitempty"`
	PaymentMethod       *string `json:"payment_method,omitempty"`
	ExternalReferenceID *string `json:"external_reference_id,omitempty"`
}

func (h *TransactionHandler) ProcessTopup(c *fiber.Ctx) error {
	account := c.Locals("account").(*models.Account)

	var req TopupRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	// Convert GSALT amount to units (1 GSALT = 100 units)
	amountGsalt, err := decimal.NewFromString(req.AmountGsalt)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}
	amountGsaltUnits := amountGsalt.Mul(decimal.NewFromInt(100)).IntPart()

	transaction, err := h.transactionService.ProcessTopup(
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

	return pkg.SuccessResponse(c, transaction)
}

type TransferRequest struct {
	DestinationAccountID string  `json:"destination_account_id" validate:"required,uuid"`
	AmountGsalt          string  `json:"amount_gsalt" validate:"required"`
	Description          *string `json:"description,omitempty"`
}

func (h *TransactionHandler) ProcessTransfer(c *fiber.Ctx) error {
	account := c.Locals("account").(*models.Account)

	var req TransferRequest
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

type PaymentRequest struct {
	AmountGsalt         string  `json:"amount_gsalt" validate:"required"`
	PaymentAmount       *int64  `json:"payment_amount,omitempty"`
	PaymentCurrency     *string `json:"payment_currency,omitempty"`
	PaymentMethod       *string `json:"payment_method,omitempty"`
	Description         *string `json:"description,omitempty"`
	ExternalReferenceID *string `json:"external_reference_id,omitempty"`
}

func (h *TransactionHandler) ProcessPayment(c *fiber.Ctx) error {
	account := c.Locals("account").(*models.Account)

	var req PaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	// Convert GSALT amount to units (1 GSALT = 100 units)
	amountGsalt, err := decimal.NewFromString(req.AmountGsalt)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}
	amountGsaltUnits := amountGsalt.Mul(decimal.NewFromInt(100)).IntPart()

	transaction, err := h.transactionService.ProcessPayment(
		account.ConnectID.String(),
		amountGsaltUnits,
		req.PaymentAmount,
		req.PaymentCurrency,
		req.PaymentMethod,
		req.Description,
		req.ExternalReferenceID,
	)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, transaction)
}
