package main

import (
	"log"
	"os"

	"github.com/hakaton/subscriptions-backend/internal/config"
	"github.com/hakaton/subscriptions-backend/internal/db"
	"github.com/hakaton/subscriptions-backend/internal/server"
)

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	srv := server.NewServer(database, cfg)

	addr := ":" + cfg.HTTPPort
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		addr = ":" + fromEnv
	}

	log.Printf("starting API server on %s", addr)

	if err = srv.Listen(addr); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}

