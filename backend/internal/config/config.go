package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort    string
	DatabaseURL string
}

// Load загружает конфиг из .env и переменных окружения.
// Для MVP этого достаточно, потом можно расширить.
func Load() Config {
	_ = godotenv.Load()

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// TODO: поменять на реальный DSN Postgres
		dbURL = "postgres://user:password@localhost:5432/subscriptions?sslmode=disable"
	}

	return Config{
		HTTPPort:    httpPort,
		DatabaseURL: dbURL,
	}
}

