package deliveries

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/safatanc/gsalt-core/internal/app/middlewares"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/app/pkg"
	"github.com/safatanc/gsalt-core/internal/app/services"
)

type PaymentHandler struct {
	paymentService *services.PaymentService
	authMiddleware *middlewares.AuthMiddleware
}

func NewPaymentHandler(paymentService *services.PaymentService, authMiddleware *middlewares.AuthMiddleware) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		authMiddleware: authMiddleware,
	}
}

func (h *PaymentHandler) RegisterRoutes(router fiber.Router) {
	paymentGroup := router.Group("/payment-accounts", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount)

	paymentGroup.Post("/", h.CreatePaymentAccount)
	paymentGroup.Get("/", h.GetPaymentAccounts)
	paymentGroup.Post("/:id/verify", h.VerifyPaymentAccount)
	paymentGroup.Delete("/:id", h.DeactivatePaymentAccount)
}

func (h *PaymentHandler) CreatePaymentAccount(c *fiber.Ctx) error {
	var req models.PaymentAccountCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	// Get account from context
	account := c.Locals("account").(*models.Account)
	req.AccountID = account.ConnectID

	paymentAccount, err := h.paymentService.CreatePaymentAccount(c.Context(), &req)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, paymentAccount)
}

func (h *PaymentHandler) GetPaymentAccounts(c *fiber.Ctx) error {
	account := c.Locals("account").(*models.Account)

	accounts, err := h.paymentService.GetPaymentAccounts(c.Context(), account.ConnectID)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, accounts)
}

func (h *PaymentHandler) VerifyPaymentAccount(c *fiber.Ctx) error {
	accountID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	err = h.paymentService.VerifyPaymentAccount(c.Context(), accountID)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse[any](c, nil)
}

func (h *PaymentHandler) DeactivatePaymentAccount(c *fiber.Ctx) error {
	accountID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	err = h.paymentService.DeactivatePaymentAccount(c.Context(), accountID)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse[any](c, nil)
}
