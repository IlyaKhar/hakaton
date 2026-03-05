package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	authjwt "github.com/hakaton/subscriptions-backend/internal/auth"
	"github.com/hakaton/subscriptions-backend/internal/config"
)

// AuthMiddleware возвращает Fiber-middleware, которое:
// 1) читает JWT токен из заголовка Authorization
// 2) валидирует JWT
// 3) кладёт userID в c.Locals("userID")
func AuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("Authorization")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "отсутствует заголовок Authorization",
			})
		}

		// Убираем пробелы
		token = strings.TrimSpace(token)

		// Если пришёл с префиксом "Bearer ", убираем его (для совместимости)
		if strings.HasPrefix(strings.ToLower(token), "bearer ") {
			token = strings.TrimSpace(token[7:])
		}

		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "токен не найден",
			})
		}

		userID, err := authjwt.ParseToken(token, cfg.JwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "некорректный или просроченный токен",
			})
		}

		c.Locals("userID", userID)

		return c.Next()
	}
}
