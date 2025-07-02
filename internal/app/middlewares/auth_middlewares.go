package middlewares

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/app/pkg"
	"github.com/safatanc/gsalt-core/internal/app/services"
)

type AuthMiddleware struct {
	connectService *services.ConnectService
	accountService *services.AccountService
}

func NewAuthMiddleware(connectService *services.ConnectService, accountService *services.AccountService) *AuthMiddleware {
	return &AuthMiddleware{connectService: connectService, accountService: accountService}
}

func (m *AuthMiddleware) AuthConnect(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(models.WebResponse[any]{
			Success: false,
			Message: "Unauthorized",
		})
	}

	token = strings.Replace(token, "Bearer ", "", 1)

	connectUser, err := m.connectService.GetCurrentUser(token)
	if err != nil {
		return pkg.ErrorResponse(c, errors.NewBadRequestError(err.Error()))
	}

	c.Locals("connect_user", connectUser)

	return c.Next()
}

func (m *AuthMiddleware) AuthAccount(c *fiber.Ctx) error {
	connectUser := c.Locals("connect_user").(*models.ConnectUser)

	if connectUser == nil {
		return pkg.ErrorResponse(c, errors.NewUnauthorizedError("User is not authenticated"))
	}

	account, err := m.accountService.GetAccount(connectUser.ID.String())
	if err != nil {
		return pkg.ErrorResponse(c, errors.NewUnauthorizedError(fmt.Sprintf("User with connect username %s is not registered on GSALT. Please register first.", connectUser.Username)))
	}

	if account.Status != models.AccountStatusActive {
		return pkg.ErrorResponse(c, errors.NewUnauthorizedError(fmt.Sprintf("User is not active (%s)", account.Status)))
	}

	c.Locals("account", account)

	return c.Next()
}

func (m *AuthMiddleware) AuthMerchant(c *fiber.Ctx) error {
	account := c.Locals("account").(*models.Account)

	if account == nil {
		return pkg.ErrorResponse(c, errors.NewUnauthorizedError("User is not authenticated"))
	}

	if account.AccountType != models.AccountTypeMerchant {
		return pkg.ErrorResponse(c, errors.NewUnauthorizedError("User is not a merchant"))
	}

	if account.KYCStatus != models.KYCStatusVerified {
		return pkg.ErrorResponse(c, errors.NewUnauthorizedError(fmt.Sprintf("User KYC is not verified (%s)", account.KYCStatus)))
	}

	return c.Next()
}

func (m *AuthMiddleware) AuthAdmin(c *fiber.Ctx) error {
	connectUser := c.Locals("connect_user").(*models.ConnectUser)

	if connectUser == nil {
		return pkg.ErrorResponse(c, errors.NewUnauthorizedError("User is not authenticated"))
	}

	if connectUser.GlobalRole != models.ConnectUserRoleAdmin {
		return pkg.ErrorResponse(c, errors.NewUnauthorizedError("User is not an admin"))
	}

	return c.Next()
}
