package deliveries

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/safatanc/gsalt-core/internal/app/middlewares"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/app/pkg"
	"github.com/safatanc/gsalt-core/internal/app/services"
)

type VoucherRedemptionHandler struct {
	voucherRedemptionService *services.VoucherRedemptionService
	authMiddleware           *middlewares.AuthMiddleware
}

func NewVoucherRedemptionHandler(voucherRedemptionService *services.VoucherRedemptionService, authMiddleware *middlewares.AuthMiddleware) *VoucherRedemptionHandler {
	return &VoucherRedemptionHandler{
		voucherRedemptionService: voucherRedemptionService,
		authMiddleware:           authMiddleware,
	}
}

func (h *VoucherRedemptionHandler) RegisterRoutes(router fiber.Router) {
	redemptionGroup := router.Group("/voucher-redemptions")

	// All endpoints require authentication
	redemptionGroup.Post("/", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.CreateRedemption)
	redemptionGroup.Get("/me", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.GetMyRedemptions)
	redemptionGroup.Get("/:id", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.GetRedemption)
	redemptionGroup.Patch("/:id", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.UpdateRedemption)
	redemptionGroup.Delete("/:id", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.DeleteRedemption)

	// Voucher redemption endpoint
	redemptionGroup.Post("/redeem", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.RedeemVoucher)

	// Admin endpoints for getting redemptions by voucher
	redemptionGroup.Get("/voucher/:voucher_id", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.GetRedemptionsByVoucher)
}

func (h *VoucherRedemptionHandler) CreateRedemption(c *fiber.Ctx) error {
	var dto models.VoucherRedemptionCreateDto
	if err := c.BodyParser(&dto); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	redemption, err := h.voucherRedemptionService.CreateRedemption(&dto)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, redemption)
}

func (h *VoucherRedemptionHandler) GetRedemption(c *fiber.Ctx) error {
	id := c.Params("id")

	redemption, err := h.voucherRedemptionService.GetRedemption(id)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, redemption)
}

func (h *VoucherRedemptionHandler) GetMyRedemptions(c *fiber.Ctx) error {
	account := c.Locals("account").(*models.Account)

	// Parse query parameters
	limitStr := c.Query("limit", "10")
	offsetStr := c.Query("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	redemptions, err := h.voucherRedemptionService.GetRedemptionsByAccount(account.ConnectID.String(), limit, offset)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, redemptions)
}

func (h *VoucherRedemptionHandler) GetRedemptionsByVoucher(c *fiber.Ctx) error {
	voucherId := c.Params("voucher_id")

	// Parse query parameters
	limitStr := c.Query("limit", "10")
	offsetStr := c.Query("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	redemptions, err := h.voucherRedemptionService.GetRedemptionsByVoucher(voucherId, limit, offset)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, redemptions)
}

func (h *VoucherRedemptionHandler) UpdateRedemption(c *fiber.Ctx) error {
	id := c.Params("id")

	var dto models.VoucherRedemptionUpdateDto
	if err := c.BodyParser(&dto); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	redemption, err := h.voucherRedemptionService.UpdateRedemption(id, &dto)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, redemption)
}

func (h *VoucherRedemptionHandler) DeleteRedemption(c *fiber.Ctx) error {
	id := c.Params("id")

	err := h.voucherRedemptionService.DeleteRedemption(id)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse[any](c, nil)
}

type RedeemVoucherRequest struct {
	VoucherCode string `json:"voucher_code" validate:"required"`
}

func (h *VoucherRedemptionHandler) RedeemVoucher(c *fiber.Ctx) error {
	account := c.Locals("account").(*models.Account)

	var req RedeemVoucherRequest
	if err := c.BodyParser(&req); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	redemption, transaction, err := h.voucherRedemptionService.RedeemVoucher(
		account.ConnectID.String(),
		req.VoucherCode,
	)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	response := map[string]interface{}{
		"redemption":  redemption,
		"transaction": transaction,
	}

	return pkg.SuccessResponse(c, response)
}
