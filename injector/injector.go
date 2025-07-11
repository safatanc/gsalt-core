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
	HealthHandler            *deliveries.HealthHandler
	AccountHandler           *deliveries.AccountHandler
	TransactionHandler       *deliveries.TransactionHandler
	VoucherHandler           *deliveries.VoucherHandler
	VoucherRedemptionHandler *deliveries.VoucherRedemptionHandler
	RateLimitMiddleware      *middlewares.RateLimitMiddleware
	APIKeyMiddleware         *middlewares.APIKeyMiddleware
}

// RegisterRoutes registers all application routes using a Fiber router
func (app *Application) RegisterRoutes(router fiber.Router) {
	// Apply global rate limit for public API
	router.Use(app.RateLimitMiddleware.LimitByIP(middlewares.PublicAPILimit))

	// Auth endpoints with stricter rate limit
	authGroup := router.Group("/auth")
	authGroup.Use(app.RateLimitMiddleware.LimitByIP(middlewares.AuthLimit))

	// Protected API endpoints with user-based rate limit
	protectedGroup := router.Group("")
	protectedGroup.Use(app.RateLimitMiddleware.LimitByUser(middlewares.AuthenticatedAPILimit))

	// Register all handlers
	app.HealthHandler.RegisterRoutes(router)
	app.AccountHandler.RegisterRoutes(router)
	app.TransactionHandler.RegisterRoutes(router)
	app.VoucherHandler.RegisterRoutes(router)
	app.VoucherRedemptionHandler.RegisterRoutes(router)
}

// Infrastructure providers
var infrastructureSet = wire.NewSet(
	infrastructures.NewDatabase,
	infrastructures.NewRedisClient,
	infrastructures.NewValidator,
	infrastructures.NewFlipClient,
	wire.Value("gsalt"),
	wire.Bind(new(middlewares.RateLimiter), new(*middlewares.RedisRateLimiter)),
	middlewares.NewRedisRateLimiter,
)

// Service providers
var serviceSet = wire.NewSet(
	services.NewConnectService,
	services.NewAccountService,
	services.NewPaymentMethodService,
	services.NewFlipService,
	services.NewTransactionService,
	services.NewVoucherService,
	services.NewVoucherRedemptionService,
	services.NewAuditService,
	services.NewMerchantAPIKeyService,
	services.NewPaymentService,
)

// Middleware providers
var middlewareSet = wire.NewSet(
	middlewares.NewAuthMiddleware,
	middlewares.NewAPIKeyMiddleware,
	middlewares.NewRateLimitMiddleware,
)

// Handler providers
var handlerSet = wire.NewSet(
	deliveries.NewHealthHandler,
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
