package server

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/hakaton/subscriptions-backend/internal/config"
	"github.com/hakaton/subscriptions-backend/internal/middleware"
	"github.com/hakaton/subscriptions-backend/internal/server/routes"
)

type Server struct {
	app *fiber.App
}

// NewServer инициализирует Fiber, middleware и основные роуты API.
func NewServer(db *sql.DB, cfg config.Config) *Server {
	app := fiber.New()

	middleware.RegisterBasicMiddleware(app)

	// Вешаем все маршруты на префикс /api/v1
	// Эндпоинты:
	//   - GET  /api/v1/health
	//   - GET  /api/v1/swagger/index.html
	//   - POST /api/v1/auth/register
	//   - POST /api/v1/auth/login
	//   - GET  /api/v1/auth/me
	apiV1 := app.Group("/api/v1")
	routes.RegisterV1Routes(apiV1, db, cfg)

	return &Server{app: app}
}

// Listen запускает HTTP-сервер.
func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}
