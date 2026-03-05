package routes

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	_ "github.com/hakaton/subscriptions-backend/docs"
	"github.com/hakaton/subscriptions-backend/internal/analytics"
	"github.com/hakaton/subscriptions-backend/internal/config"
	"github.com/hakaton/subscriptions-backend/internal/dashboard"
	"github.com/hakaton/subscriptions-backend/internal/forecast"
	"github.com/hakaton/subscriptions-backend/internal/middleware"
	"github.com/hakaton/subscriptions-backend/internal/notifications"
	"github.com/hakaton/subscriptions-backend/internal/payment_cards"
	"github.com/hakaton/subscriptions-backend/internal/recommendations"
	"github.com/hakaton/subscriptions-backend/internal/server/routes/v1/auth"
	"github.com/hakaton/subscriptions-backend/internal/sources"
	"github.com/hakaton/subscriptions-backend/internal/subscriptions"
	"github.com/hakaton/subscriptions-backend/internal/transactions"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

// RegisterV1Routes регистрирует все v1-роуты API.
func RegisterV1Routes(r fiber.Router, db *sql.DB, cfg config.Config) {
	// health-check
	r.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// swagger UI доступен по /api/v1/swagger/index.html
	r.Get("/swagger/*", fiberSwagger.WrapHandler)

	// auth (без JWT-мидлвари для register/login/forgot-password/verify-code/reset-password)
	authGroup := r.Group("/auth")
	auth.RegisterPublicRoutes(authGroup, db, &cfg)

	// Группа защищённых роутов, к которым нужен JWT
	protected := r.Group("", middleware.AuthMiddleware(&cfg))

	// /auth/me требует авторизацию
	auth.RegisterProtectedRoutes(protected.Group("/auth"), db, &cfg)

	// subscriptions
	subscriptions.RegisterRoutes(protected.Group("/subscriptions"), db)

	// sources
	// публичный OAuth callback (без JWT)
	sources.RegisterOAuthCallbackRoutes(r.Group("/sources/oauth"), db, &cfg)
	// защищённые роуты sources (включая /oauth/:provider/authorize)
	sources.RegisterRoutes(protected.Group("/sources"), db, &cfg)

	// analytics
	analytics.RegisterRoutes(protected.Group("/analytics"), db)

	// dashboard
	dashboard.RegisterRoutes(protected.Group("/dashboard"), db)

	// forecast
	forecast.RegisterRoutes(protected.Group("/forecast"), db)

	// notifications
	notifications.RegisterRoutes(protected.Group("/notifications"), db)

	// recommendations (зависят от userId для персональных рекомендаций, поэтому тоже под auth)
	recommendations.RegisterRoutes(protected.Group("/recommendations"), db)

	// payment-cards
	payment_cards.RegisterRoutes(protected.Group("/payment-cards"), db)

	// transactions
	transactions.RegisterRoutes(protected.Group("/transactions"), db)
}
