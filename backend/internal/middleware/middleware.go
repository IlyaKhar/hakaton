package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterBasicMiddleware вешает базовые middleware для всего приложения.
// Здесь потом можно добавить логирование, recover, CORS и т.п.
func RegisterBasicMiddleware(app *fiber.App) {
	app.Use(func(c *fiber.Ctx) error {
		// Простой health-check middleware/лог для MVP (можно расширить)
		return c.Next()
	})
}
