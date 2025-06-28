package deliveries

import (
	"github.com/gofiber/fiber/v2"
	"github.com/safatanc/gsalt-core/internal/app/middlewares"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/app/pkg"
	"github.com/safatanc/gsalt-core/internal/app/services"
)

type AccountHandler struct {
	accountService *services.AccountService
	authMiddleware *middlewares.AuthMiddleware
}

func NewAccountHandler(accountService *services.AccountService, authMiddleware *middlewares.AuthMiddleware) *AccountHandler {
	return &AccountHandler{accountService: accountService, authMiddleware: authMiddleware}
}

func (h *AccountHandler) RegisterRoutes(router fiber.Router) {
	accountGroup := router.Group("/accounts")

	accountGroup.Post("/", h.authMiddleware.AuthConnect, h.CreateAccount)
	accountGroup.Get("/me", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.GetMe)
	accountGroup.Delete("/me", h.authMiddleware.AuthConnect, h.authMiddleware.AuthAccount, h.DeleteMe)
	accountGroup.Get("/:id", h.GetAccountByID)
}

func (h *AccountHandler) CreateAccount(c *fiber.Ctx) error {
	accessToken := c.Get("Authorization")

	account, err := h.accountService.CreateAccount(accessToken)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, account)
}

func (h *AccountHandler) GetMe(c *fiber.Ctx) error {
	account := c.Locals("account").(*models.Account)
	return pkg.SuccessResponse(c, account)
}

func (h *AccountHandler) GetAccountByID(c *fiber.Ctx) error {
	id := c.Params("id")

	account, err := h.accountService.GetAccount(id)
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse(c, account)
}

func (h *AccountHandler) DeleteMe(c *fiber.Ctx) error {
	account := c.Locals("account").(*models.Account)

	err := h.accountService.DeleteAccount(account.ConnectID.String())
	if err != nil {
		return pkg.ErrorResponse(c, err)
	}

	return pkg.SuccessResponse[any](c, nil)
}
