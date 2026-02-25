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
func NewServer(_ *sql.DB, cfg config.Config) *Server {
	app := fiber.New()

	middleware.RegisterBasicMiddleware(app)

	api := app.Group("/api")
	v1 := api.Group("/v1")

	routes.RegisterV1Routes(v1, cfg)

	return &Server{app: app}
}

// Listen запускает HTTP-сервер.
func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}

