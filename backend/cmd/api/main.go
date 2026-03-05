// @title           Subscriptions API
// @version         1.0
// @description     API для управления подписками пользователей
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Введите JWT токен

package main

import (
	"fmt"
	"log"

	"github.com/hakaton/subscriptions-backend/docs"
	"github.com/hakaton/subscriptions-backend/internal/config"
	"github.com/hakaton/subscriptions-backend/internal/db"
	"github.com/hakaton/subscriptions-backend/internal/server"
)

func main() {
	cfg := config.Load()

	// ВРЕМЕННО: логируем, что реально подхватилось из .env
	log.Printf("DB config: host=%s port=%s user=%s db=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBName,
	)

	// Настраиваем swagger: все ручки живут под /api/v1
	docs.SwaggerInfo.BasePath = "/api/v1"

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
	)

	database, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	srv := server.NewServer(database, *cfg)

	addr := ":" + cfg.Port

	log.Printf("starting API server on %s", addr)

	if err = srv.Listen(addr); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
