//go:build wireinject
// +build wireinject

package injector

import (
	"github.com/gofiber/fiber/v2" // Add Fiber import if you're using it
	"github.com/google/wire"
	"github.com/safatanc/gsalt-core/internal/app/deliveries"
	"github.com/safatanc/gsalt-core/internal/app/middlewares"
	"github.com/safatanc/gsalt-core/internal/app/services"
	"github.com/safatanc/gsalt-core/internal/infrastructures"
)

// Application represents the main application container for gsalt-core
type Application struct {
	AccountHandler           *deliveries.AccountHandler
	TransactionHandler       *deliveries.TransactionHandler
	VoucherHandler           *deliveries.VoucherHandler
	VoucherRedemptionHandler *deliveries.VoucherRedemptionHandler
}

// RegisterRoutes registers all application routes using a Fiber router
func (app *Application) RegisterRoutes(router fiber.Router) {
	// You'll need to implement these RegisterRoutes methods within your handlers
	// For example, in deliveries/account_handler.go, you'd have:
	// func (h *AccountHandler) RegisterRoutes(router fiber.Router) {
	//     router.Get("/accounts", h.GetAllAccounts)
	//     router.Post("/accounts", h.CreateAccount)
	// }
	app.AccountHandler.RegisterRoutes(router)
	app.TransactionHandler.RegisterRoutes(router)
	app.VoucherHandler.RegisterRoutes(router)
	app.VoucherRedemptionHandler.RegisterRoutes(router)
}

// Infrastructure providers
var infrastructureSet = wire.NewSet(
	infrastructures.NewDatabase,
	infrastructures.NewValidator,
)

// Service providers
var serviceSet = wire.NewSet(
	services.NewConnectService,
	services.NewAccountService,
	services.NewTransactionService,
	services.NewVoucherService,
	services.NewVoucherRedemptionService,
)

// Middleware providers
var middlewareSet = wire.NewSet(
	middlewares.NewAuthMiddleware,
)

// Handler providers
var handlerSet = wire.NewSet(
	deliveries.NewAccountHandler,
	deliveries.NewTransactionHandler,
	deliveries.NewVoucherHandler,
	deliveries.NewVoucherRedemptionHandler,
	wire.Struct(new(Application), "*"), // This tells Wire to build the Application struct
)

// InitializeApplication initializes the application with all its dependencies
func InitializeApplication() (*Application, error) {
	wire.Build(
		infrastructureSet,
		serviceSet,
		middlewareSet,
		handlerSet,
	)
	return &Application{}, nil // Wire will populate the Application struct based on handlerSet
}
