package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	JwtSecret     string
	RefreshSecret string
	BaseUrl       string
	// SMTP настройки для отправки email
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string // email отправителя (например, noreply@example.com)
	// OAuth настройки для почтовых провайдеров
	MailruClientID     string
	MailruClientSecret string
	YandexClientID     string
	YandexClientSecret string
	GmailClientID      string
	GmailClientSecret  string
}

func Load() *Config {
	godotenv.Load()

	return &Config{
		Port:               LoadEnv("PORT", "3000"),
		DBHost:             LoadEnv("DB_HOST", "localhost"),
		DBPort:             LoadEnv("DB_PORT", "5432"),
		DBUser:             LoadEnv("DB_USER", "postgres"),
		DBPassword:         LoadEnv("DB_PASSWORD", ""),
		DBName:             LoadEnv("DB_NAME", "postgres"),
		JwtSecret:          LoadEnv("JWT_SECRET", "not found"),
		RefreshSecret:      LoadEnv("REFRESH_SECRET", "not found"),
		BaseUrl:            LoadEnv("BASE_URL", "http://localhost:3000"),
		SMTPHost:           LoadEnv("SMTP_HOST", ""),
		SMTPPort:           LoadEnv("SMTP_PORT", "465"),
		SMTPUser:           LoadEnv("SMTP_USER", ""),
		SMTPPassword:       LoadEnv("SMTP_PASSWORD", ""),
		SMTPFrom:           LoadEnv("SMTP_FROM", ""),
		MailruClientID:     LoadEnv("MAILRU_CLIENT_ID", ""),
		MailruClientSecret: LoadEnv("MAILRU_CLIENT_SECRET", ""),
		YandexClientID:     LoadEnv("YANDEX_CLIENT_ID", ""),
		YandexClientSecret: LoadEnv("YANDEX_CLIENT_SECRET", ""),
		GmailClientID:      LoadEnv("GMAIL_CLIENT_ID", ""),
		GmailClientSecret:  LoadEnv("GMAIL_CLIENT_SECRET", ""),
	}
}

func LoadEnv(key string, replacement string) string {
	res := os.Getenv(key)

	if res == "" {
		return replacement
	}

	return res
}
