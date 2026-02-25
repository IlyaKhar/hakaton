package auth

import "github.com/gofiber/fiber/v2"

// RegisterRoutes регистрирует auth-роуты.
// Сейчас это заглушки, чтобы у тебя был скелет под реализацию.
func RegisterRoutes(r fiber.Router) {
	r.Post("/register", registerHandler)
	r.Post("/login", loginHandler)
	r.Get("/me", meHandler)
}

func registerHandler(c *fiber.Ctx) error {
	// TODO: реализовать регистрацию пользователя
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"message": "not implemented"})
}

func loginHandler(c *fiber.Ctx) error {
	// TODO: реализовать логин пользователя и выдачу JWT
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"message": "not implemented"})
}

func meHandler(c *fiber.Ctx) error {
	// TODO: вернуть текущего пользователя по JWT
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"message": "not implemented"})
}

