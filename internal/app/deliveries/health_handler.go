package deliveries

import (
	"github.com/gofiber/fiber/v2"
	"github.com/safatanc/gsalt-core/internal/app/pkg"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) RegisterRoutes(router fiber.Router) {
	router.Get("/health", h.GetHealth)
}

func (h *HealthHandler) GetHealth(c *fiber.Ctx) error {
	return pkg.SuccessResponse(c, "gsalt-core")
}
