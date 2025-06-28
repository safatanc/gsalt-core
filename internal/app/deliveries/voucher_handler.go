package deliveries

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/safatanc/gsalt-core/internal/app/middlewares"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/app/pkg"
	"github.com/safatanc/gsalt-core/internal/app/services"
)

type VoucherHandler struct {
	voucherService *services.VoucherService
	authMiddleware *middlewares.AuthMiddleware
}

func NewVoucherHandler(voucherService *services.VoucherService, authMiddleware *middlewares.AuthMiddleware) *VoucherHandler {
	return &VoucherHandler{
		voucherService: voucherService,
		authMiddleware: authMiddleware,
	}
}

func (h *VoucherHandler) RegisterRoutes(router fiber.Router) {
	voucherGroup := router.Group("/vouchers")

	// Public endpoints
	voucherGroup.Get("/", h.GetVouchers)
	voucherGroup.Get("/:id", h.GetVoucher)
	voucherGroup.Get("/code/:code", h.GetVoucherByCode)
	voucherGroup.Post("/validate/:code", h.ValidateVoucher)

	// Protected endpoints (require authentication)
	voucherGroup.Post("/", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.CreateVoucher)
	voucherGroup.Patch("/:id", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.UpdateVoucher)
	voucherGroup.Delete("/:id", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.DeleteVoucher)
}

func (h *VoucherHandler) CreateVoucher(c *fiber.Ctx) error {
	var dto models.VoucherCreateDto
	if err := c.BodyParser(&dto); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	// Set created by from authenticated account
	account := c.Locals("account").(*models.Account)
	createdBy := account.ConnectID.String()
	dto.CreatedBy = &createdBy

	voucher, err := h.voucherService.CreateVoucher(&dto)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, voucher)
}

func (h *VoucherHandler) GetVoucher(c *fiber.Ctx) error {
	id := c.Params("id")

	voucher, err := h.voucherService.GetVoucher(id)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, voucher)
}

func (h *VoucherHandler) GetVoucherByCode(c *fiber.Ctx) error {
	code := c.Params("code")

	voucher, err := h.voucherService.GetVoucherByCode(code)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, voucher)
}

func (h *VoucherHandler) GetVouchers(c *fiber.Ctx) error {
	// Parse query parameters
	limitStr := c.Query("limit", "10")
	offsetStr := c.Query("offset", "0")
	statusStr := c.Query("status")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	var status *models.VoucherStatus
	if statusStr != "" {
		voucherStatus := models.VoucherStatus(statusStr)
		status = &voucherStatus
	}

	vouchers, err := h.voucherService.GetVouchers(limit, offset, status)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, vouchers)
}

func (h *VoucherHandler) UpdateVoucher(c *fiber.Ctx) error {
	id := c.Params("id")

	var dto models.VoucherUpdateDto
	if err := c.BodyParser(&dto); err != nil {
		return pkg.ErrorResponse(c, err)
	}

	voucher, err := h.voucherService.UpdateVoucher(id, &dto)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, voucher)
}

func (h *VoucherHandler) DeleteVoucher(c *fiber.Ctx) error {
	id := c.Params("id")

	err := h.voucherService.DeleteVoucher(id)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse[any](c, nil)
}

func (h *VoucherHandler) ValidateVoucher(c *fiber.Ctx) error {
	code := c.Params("code")

	voucher, err := h.voucherService.ValidateVoucher(code)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	response := map[string]interface{}{
		"valid":   true,
		"voucher": voucher,
	}

	return pkg.SuccessResponse(c, response)
}
