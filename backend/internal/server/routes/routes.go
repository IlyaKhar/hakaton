package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/hakaton/subscriptions-backend/internal/config"
	"github.com/hakaton/subscriptions-backend/internal/server/routes/v1/auth"
)

// RegisterV1Routes регистрирует все v1-роуты API.
func RegisterV1Routes(r fiber.Router, cfg config.Config) {
	_ = cfg // пока не используется, но пригодится позже

	// health-check
	r.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// auth
	auth.RegisterRoutes(r.Group("/auth"))

	// TODO: здесь же потом повесить:
	// - /subscriptions
	// - /sources
	// - /analytics
	// - /forecast
	// - /notifications
	// - /recommendations
}

