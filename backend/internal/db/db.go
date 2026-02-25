package db

import (
	"database/sql"

	_ "github.com/lib/pq"
)

// Connect создаёт подключение к Postgres по переданному DSN.
func Connect(dsn string) (*sql.DB, error) {
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err = database.Ping(); err != nil {
		return nil, err
	}

	return database, nil
}
