package common

import "github.com/gofiber/fiber/v2"

// ErrorResponse базовый формат ошибки.
type ErrorResponse struct {
	Message string `json:"message"`
}

// JSONError — хелпер для возврата ошибки в едином формате.
func JSONError(c *fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(ErrorResponse{Message: msg})
}
